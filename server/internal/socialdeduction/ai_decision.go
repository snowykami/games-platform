package socialdeduction

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

type staleAIDecisionError struct {
	RoomID            string
	PlayerID          string
	PlayerName        string
	ExpectedPhase     Phase
	CurrentPhase      Phase
	ExpectedUpdatedAt time.Time
	CurrentUpdatedAt  time.Time
	PlayerFound       bool
	PlayerAlive       bool
	PlayerIsAI        bool
	ActionID          string
	Reason            string
}

func (err staleAIDecisionError) Error() string {
	return fmt.Sprintf("stale_ai_decision reason=%s expectedPhase=%s currentPhase=%s expectedUpdatedAt=%s currentUpdatedAt=%s playerFound=%t playerAlive=%t playerIsAI=%t actionID=%s",
		err.Reason,
		err.ExpectedPhase,
		err.CurrentPhase,
		err.ExpectedUpdatedAt.Format(time.RFC3339Nano),
		err.CurrentUpdatedAt.Format(time.RFC3339Nano),
		err.PlayerFound,
		err.PlayerAlive,
		err.PlayerIsAI,
		err.ActionID,
	)
}

type socialDecisionScope string

const (
	socialDecisionScopeRule   socialDecisionScope = "rule"
	socialDecisionScopeSpeech socialDecisionScope = "speech"
)

func (m *Manager) socialDecision(room *Room, player *Player, state map[string]any, actions []aiplayer.LegalAction) (aiplayer.Decision, error) {
	return m.socialDecisionScoped(room, player, state, actions, socialDecisionScopeRule)
}

func (m *Manager) socialSpeechDecision(room *Room, player *Player, state map[string]any, actions []aiplayer.LegalAction) (aiplayer.Decision, error) {
	return m.socialDecisionScoped(room, player, state, actions, socialDecisionScopeSpeech)
}

func (m *Manager) socialDecisionScoped(room *Room, player *Player, state map[string]any, actions []aiplayer.LegalAction, scope socialDecisionScope) (aiplayer.Decision, error) {
	if !m.canUseLLM(player) {
		return aiplayer.Decision{}, errors.New("llm_not_configured")
	}
	if m.aiController == nil || !m.aiController.Enabled() {
		return aiplayer.Decision{}, aiagent.ErrLLMNotConfigured
	}
	session := m.ensureAISession(room, player)
	applySocialAIGuidance(state, room, player, scope)
	state["aiSession"] = map[string]any{
		"sessionId": sessionID(room, player),
		"memory":    append([]string{}, session.Memory...),
	}
	state["privateNotes"] = aliasPlayerNotes(room, room.PlayerNotes[player.ID])
	phase := room.Phase
	ruleUpdatedAt := decisionRuleUpdatedAt(room)
	speechUpdatedAt := decisionSpeechUpdatedAt(room)
	playerID := player.ID
	playerName := player.Name
	eventType := gameactor.AgentRequiredAction
	if scope == socialDecisionScopeSpeech {
		eventType = gameactor.AgentOptionalSpeech
	}
	startedAt := time.Now()
	personality := ""
	speechStyle := ""
	if player.AI != nil {
		personality = player.AI.Personality
		speechStyle = player.AI.SpeechStyle
	}
	decision, err := m.aiController.Decide(aiagent.DecisionRequest{
		RoomID:        room.ID,
		PlayerID:      player.ID,
		RequestPrefix: "social",
		SessionID:     sessionID(room, player),
		Phase:         string(room.Phase),
		Type:          eventType,
		Profile: aiagent.Profile{
			Name:        fmt.Sprintf("座位 %d", aiPlayerNumber(room, player)),
			Personality: personality,
			SpeechStyle: speechStyle,
		},
		State:   state,
		Actions: actions,
		Unlock:  m.mu.Unlock,
		Lock:    m.mu.Lock,
		Stale: func(decision aiplayer.Decision) error {
			currentPlayer := findPlayerByID(room, playerID)
			currentRuleUpdatedAt := decisionRuleUpdatedAt(room)
			currentSpeechUpdatedAt := decisionSpeechUpdatedAt(room)
			staleByRule := !currentRuleUpdatedAt.Equal(ruleUpdatedAt)
			staleBySpeech := scope == socialDecisionScopeSpeech && !currentSpeechUpdatedAt.Equal(speechUpdatedAt)
			if socialAIDecisionPlayerCanAct(room, currentPlayer, phase, scope) && !staleByRule && !staleBySpeech {
				return nil
			}
			expectedUpdatedAt := ruleUpdatedAt
			currentUpdatedAt := currentRuleUpdatedAt
			if !staleByRule && staleBySpeech {
				expectedUpdatedAt = speechUpdatedAt
				currentUpdatedAt = currentSpeechUpdatedAt
			}
			staleErr := staleAIDecisionError{
				RoomID:            room.ID,
				PlayerID:          playerID,
				PlayerName:        playerName,
				ExpectedPhase:     phase,
				CurrentPhase:      room.Phase,
				ExpectedUpdatedAt: expectedUpdatedAt,
				CurrentUpdatedAt:  currentUpdatedAt,
				PlayerFound:       currentPlayer != nil,
				ActionID:          decision.ActionID,
				Reason:            staleReason(room, currentPlayer, phase, ruleUpdatedAt, speechUpdatedAt, scope),
			}
			if currentPlayer != nil {
				staleErr.PlayerAlive = currentPlayer.Alive
				staleErr.PlayerIsAI = currentPlayer.IsAI
			}
			lastSpeechID := ""
			lastSpeechPlayer := ""
			if len(room.Speeches) > 0 {
				lastSpeech := room.Speeches[len(room.Speeches)-1]
				lastSpeechID = lastSpeech.ID
				lastSpeechPlayer = lastSpeech.PlayerName
			}
			slog.Warn("social llm decision became stale",
				"room", room.ID,
				"game", room.Game,
				"player", playerID,
				"playerName", playerName,
				"reason", staleErr.Reason,
				"expectedPhase", phase,
				"currentPhase", room.Phase,
				"scope", scope,
				"expectedRuleUpdatedAt", ruleUpdatedAt,
				"currentRuleUpdatedAt", currentRuleUpdatedAt,
				"expectedSpeechUpdatedAt", speechUpdatedAt,
				"currentSpeechUpdatedAt", currentSpeechUpdatedAt,
				"playerFound", staleErr.PlayerFound,
				"playerAlive", staleErr.PlayerAlive,
				"playerIsAI", staleErr.PlayerIsAI,
				"actionID", decision.ActionID,
				"reasonLength", len(decision.Reason),
				"speechLength", len(decision.Speech),
				"lastSpeechID", lastSpeechID,
				"lastSpeechPlayer", lastSpeechPlayer,
				"duration", time.Since(startedAt),
			)
			return staleErr
		},
	})
	duration := time.Since(startedAt)
	currentPlayer := findPlayerByID(room, playerID)
	if err != nil {
		recordAIDebugTrace(room, currentPlayer, phase, scope, actions, decision, err, duration)
		return decision, err
	}
	recordAIDebugTrace(room, currentPlayer, phase, scope, actions, decision, nil, duration)
	m.applyAINotes(room, currentPlayer, decision.Notes)
	m.rememberAI(room, currentPlayer, fmt.Sprintf("phase=%s action=%s reason=%s speech=%s", room.Phase, decision.ActionID, strings.TrimSpace(decision.Reason), strings.TrimSpace(decision.Speech)))
	return decision, nil
}

func staleReason(room *Room, player *Player, expectedPhase Phase, expectedRuleUpdatedAt time.Time, expectedSpeechUpdatedAt time.Time, scope socialDecisionScope) string {
	reasons := []string{}
	if player == nil {
		reasons = append(reasons, "player_missing")
	} else {
		if !player.Alive && !socialAIDecisionAllowsDeadPlayer(room, player, expectedPhase, scope) {
			reasons = append(reasons, "player_not_alive")
		}
		if !player.IsAI {
			reasons = append(reasons, "player_not_ai")
		}
	}
	if room.Phase != expectedPhase {
		reasons = append(reasons, "phase_changed")
	}
	if !decisionRuleUpdatedAt(room).Equal(expectedRuleUpdatedAt) {
		reasons = append(reasons, "rule_updated")
	}
	if scope == socialDecisionScopeSpeech && !decisionSpeechUpdatedAt(room).Equal(expectedSpeechUpdatedAt) {
		reasons = append(reasons, "speech_updated")
	}
	if len(reasons) == 0 {
		return "unknown"
	}
	return strings.Join(reasons, ",")
}

func socialAIDecisionPlayerCanAct(room *Room, player *Player, expectedPhase Phase, scope socialDecisionScope) bool {
	if room == nil || player == nil || !player.IsAI || room.Phase != expectedPhase {
		return false
	}
	return player.Alive || socialAIDecisionAllowsDeadPlayer(room, player, expectedPhase, scope)
}

func socialAIDecisionAllowsDeadPlayer(room *Room, player *Player, expectedPhase Phase, scope socialDecisionScope) bool {
	return scope == socialDecisionScopeRule &&
		expectedPhase == PhaseWerewolfHunter &&
		room.Game == GameWerewolf &&
		room.Werewolf.HunterPendingID == player.ID &&
		player.Role == RoleHunter
}

func (m *Manager) applyAINotes(room *Room, player *Player, notes map[string]string) {
	if player == nil {
		return
	}
	for targetID, note := range notes {
		target := findPlayerByID(room, targetID)
		if target == nil {
			target = playerFromAIRef(room, targetID)
		}
		if target == nil {
			continue
		}
		setPlayerNote(room, player.ID, target.ID, note)
	}
}

func (m *Manager) removeSocialAgent(roomID string, playerID string) {
	if m.aiController != nil {
		m.aiController.Remove(roomID, playerID)
	}
	delete(m.aiSessions, socialAISessionKey(roomID, playerID))
}

func (m *Manager) removeRoomAgents(roomID string) {
	if m.aiController != nil {
		m.aiController.RemoveRoom(roomID)
	}
	for key, session := range m.aiSessions {
		if session.RoomID == roomID {
			delete(m.aiSessions, key)
		}
	}
}

func (m *Manager) aiSpeechState(room *Room, player *Player) map[string]any {
	if room.Game == GameWerewolf {
		state := werewolfAIState(room, player)
		state["speechGuide"] = "像真实狼人杀玩家一样自然短句回应。lastNight 和 publicFacts 是公开且权威的游戏事实；如果 lastNight 写着有人在夜晚出局，就绝不能说平安夜、没人死、狼没动刀或被挡了；如果 publicFacts 写着某人已公开翻牌为白痴，就不能怀疑这是自爆或伪装。不要泄露隐藏身份、验人、用药、守护或狼队信息。"
		return state
	}
	return map[string]any{
		"phase":        room.Phase,
		"role":         player.Role,
		"alignment":    player.Alignment,
		"recentSpeech": aiSpeeches(room),
		"speechGuide":  "像真实玩家一样自然短句回应，可以观察、质疑、接话；没有必要说话就跳过。不要泄露隐藏身份或秘密词。",
	}
}

func werewolfPublicFacts(room *Room) []string {
	facts := []string{}
	if room.Werewolf.LastNight != "" {
		facts = append(facts, "昨夜公开结果："+aliasPlayerNamesInText(room, room.Werewolf.LastNight))
	}
	revealedIdiots := []string{}
	outPlayers := []string{}
	alivePlayers := []string{}
	for _, player := range playersBySeat(room) {
		label := fmt.Sprintf("%s:座位 %d", aiPlayerRef(room, player), aiPlayerNumber(room, player))
		if room.Werewolf.RevealedIdiots[player.ID] {
			revealedIdiots = append(revealedIdiots, fmt.Sprintf("%s 已公开翻牌为白痴，因规则免疫本次放逐，仍然存活；这不是自爆或伪装。", label))
		}
		if player.Alive {
			alivePlayers = append(alivePlayers, label)
		} else {
			outPlayers = append(outPlayers, label)
		}
	}
	facts = append(facts, revealedIdiots...)
	if len(outPlayers) > 0 {
		facts = append(facts, "已出局玩家："+strings.Join(outPlayers, "、"))
	}
	if len(alivePlayers) > 0 {
		facts = append(facts, "存活玩家："+strings.Join(alivePlayers, "、"))
	}
	return facts
}

func (m *Manager) rememberAI(room *Room, player *Player, event string) {
	if player == nil {
		return
	}
	session := m.ensureAISession(room, player)
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}
	session.Memory = append(session.Memory, event)
	if len(session.Memory) > 24 {
		session.Memory = session.Memory[len(session.Memory)-24:]
	}
}

func sessionID(room *Room, player *Player) string {
	return fmt.Sprintf("social:%s:%s:%s", room.Game, room.ID, aiPlayerRef(room, player))
}

func socialAISessionKey(roomID string, playerID string) string {
	return roomID + ":" + playerID
}
