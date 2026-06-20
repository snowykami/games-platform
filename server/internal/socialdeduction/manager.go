package socialdeduction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/roommeta"
)

const (
	werewolfMinPlayers   = 6
	werewolfMaxPlayers   = 12
	avalonMinPlayers     = 5
	avalonMaxPlayers     = 10
	undercoverMinPlayers = 4
	undercoverMaxPlayers = 10
)

type Manager struct {
	aiProvider aiplayer.Provider
	game       GameKind
	mu         sync.Mutex
	rooms      map[string]*Room
	aiSessions map[string]*socialAISession
}

func NewManager(game GameKind, aiProvider aiplayer.Provider) *Manager {
	return &Manager{aiProvider: aiProvider, aiSessions: map[string]*socialAISession{}, game: game, rooms: map[string]*Room{}}
}

func (m *Manager) CreateRoom(user UserView) PublicRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	room := &Room{
		ID:         createRoomID(m.game),
		Game:       m.game,
		HostUserID: user.ID,
		Phase:      PhaseLobby,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	room.Players = append(room.Players, createHumanPlayer(user, "host", 0))
	if room.Game == GameWerewolf {
		applyDefaultWerewolfConfig(room)
	}
	if room.Game == GameUndercover {
		applyDefaultUndercoverConfig(room)
	}
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 创建了房间。", user.DisplayName)))
	m.rooms[room.ID] = room
	return m.publicRoom(room, user.ID)
}

func (m *Manager) JoinRoom(roomID string, user UserView) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}

	player := findPlayerByUserID(room, user.ID)
	if player == nil {
		if room.Phase != PhaseLobby {
			return PublicRoom{}, errors.New("game_already_started")
		}
		if len(room.Players) >= m.maxPlayers() {
			return PublicRoom{}, errors.New("room_full")
		}
		player = createHumanPlayer(user, "player", len(room.Players))
		room.Players = append(room.Players, player)
		reconcileLobbyConfig(room)
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 加入了房间。", user.DisplayName)))
	}

	player.Connected = true
	player.DisconnectedAt = nil
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, user.ID), nil
}

func (m *Manager) Leave(roomID string, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	if player := findPlayerByUserID(room, userID); player != nil {
		player.Connected = false
		if !player.IsAI {
			player.DisconnectedAt = &now
		}
	}
}

func (m *Manager) AddAI(roomID string, actorID string, options AIOptions) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_add_ai")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("ai_only_lobby")
	}
	if len(room.Players) >= m.maxPlayers() {
		return PublicRoom{}, errors.New("room_full")
	}
	if m.aiProvider == nil || !m.aiProvider.Enabled() {
		return PublicRoom{}, errors.New("llm_not_configured")
	}

	profile := nextAIProfile(room, aiplayer.LevelLLM)
	player := &Player{
		ID:        "ai_" + randomToken(8),
		UserID:    "ai_" + randomToken(8),
		Name:      profile.Name,
		Seat:      len(room.Players),
		RoomRole:  "player",
		Kind:      "ai",
		IsAI:      true,
		Connected: true,
		Alive:     true,
		AI:        &profile,
		JoinedAt:  time.Now().UTC(),
	}
	room.Players = append(room.Players, player)
	m.ensureAISession(room, player)
	reconcileLobbyConfig(room)
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 加入了房间。", profile.Name)))
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UpdateAI(roomID string, actorID string, playerID string, options AIOptions) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_add_ai")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("ai_only_lobby")
	}
	player := findPlayerByID(room, playerID)
	if player == nil || !player.IsAI || player.AI == nil {
		return PublicRoom{}, errors.New("ai_player_not_found")
	}
	player.AI.Level = string(aiplayer.LevelLLM)
	m.ensureAISession(room, player)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) RemovePlayer(roomID string, actorID string, playerID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_remove_player")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("remove_player_only_lobby")
	}
	for index, player := range room.Players {
		if player.ID != playerID {
			continue
		}
		if player.UserID == room.HostUserID || player.RoomRole == "host" {
			return PublicRoom{}, errors.New("cannot_remove_host")
		}
		room.Players = append(room.Players[:index], room.Players[index+1:]...)
		for seat, nextPlayer := range room.Players {
			nextPlayer.Seat = seat
		}
		reconcileLobbyConfig(room)
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 被房主移出了房间。", player.Name)))
		room.UpdatedAt = time.Now().UTC()
		return m.publicRoom(room, actorID), nil
	}
	return PublicRoom{}, errors.New("player_not_found")
}

func (m *Manager) Say(roomID string, actorID string, text string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if !recordSpeech(room, player, text) {
		return PublicRoom{}, errors.New("invalid_speech")
	}
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) RenamePlayer(roomID string, actorID string, displayName string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	nextName, err := roommeta.NormalizeDisplayName(displayName)
	if err != nil {
		return PublicRoom{}, err
	}
	oldName := player.Name
	player.Name = nextName
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 改名为 %s。", oldName, nextName)))
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UpdatePlayerNote(roomID string, actorID string, targetID string, note string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	viewer := findPlayerByUserID(room, actorID)
	if viewer == nil || viewer.IsAI {
		return PublicRoom{}, errors.New("not_in_room")
	}
	if findPlayerByID(room, targetID) == nil {
		return PublicRoom{}, errors.New("player_not_found")
	}
	setPlayerNote(room, viewer.ID, targetID, note)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UpdateWerewolfRoles(roomID string, actorID string, config WerewolfRoleConfig) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameWerewolf {
		return PublicRoom{}, errors.New("not_werewolf_room")
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_update_roles")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("roles_only_lobby")
	}

	nextConfig, err := normalizeWerewolfConfig(config, len(room.Players))
	if err != nil {
		return PublicRoom{}, err
	}
	room.Werewolf.RoleConfig = nextConfig
	room.Log = append(room.Log, createLog(fmt.Sprintf("房主将角色组调整为：%s。", nextConfig.Name)))
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UpdateUndercoverConfig(roomID string, actorID string, presetID string, includeBlank bool) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameUndercover {
		return PublicRoom{}, errors.New("not_undercover_room")
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_update_undercover")
	}
	if room.Phase != PhaseLobby {
		return PublicRoom{}, errors.New("undercover_config_only_lobby")
	}
	if !undercoverPresetExists(presetID) {
		return PublicRoom{}, errors.New("invalid_undercover_preset")
	}
	room.Undercover.PresetID = presetID
	room.Undercover.IncludeBlank = includeBlank
	room.Undercover.Presets = undercoverPresets()
	room.Log = append(room.Log, createLog(fmt.Sprintf("房主选择了题库：%s。", undercoverPresetName(presetID))))
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) Start(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_start")
	}
	if room.Phase != PhaseLobby && room.Phase != PhaseFinished {
		return PublicRoom{}, errors.New("game_already_started")
	}
	if len(room.Players) < m.minPlayers() {
		return PublicRoom{}, fmt.Errorf("need_%d_players", m.minPlayers())
	}

	resetRoom(room)
	if m.game == GameWerewolf {
		startWerewolf(room)
	} else if m.game == GameAvalon {
		startAvalon(room)
	} else {
		startUndercover(room)
	}
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UndercoverDescribe(roomID string, actorID string, text string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireUndercoverActor(roomID, actorID, PhaseUndercoverDescribe)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Undercover.CurrentSpeakerID != player.ID {
		return PublicRoom{}, errors.New("not_current_speaker")
	}
	if !recordSpeech(room, player, text) {
		return PublicRoom{}, errors.New("invalid_speech")
	}
	room.Undercover.Described[player.ID] = true
	recordAction(room, PublicAction{Type: "undercover_describe", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 完成了描述。", player.Name)})
	advanceUndercoverSpeaker(room)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) UndercoverVote(roomID string, actorID string, targetID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireUndercoverActor(roomID, actorID, PhaseUndercoverVote)
	if err != nil {
		return PublicRoom{}, err
	}
	target := findPlayerByID(room, targetID)
	if target == nil || !target.Alive || target.ID == player.ID {
		return PublicRoom{}, errors.New("invalid_target")
	}
	room.Undercover.Votes[player.ID] = target.ID
	recordAction(room, PublicAction{Type: "undercover_vote", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 已投票。", player.Name)})
	resolveUndercoverVote(room)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) NightAction(roomID string, actorID string, actionID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireWerewolfActor(roomID, actorID, PhaseWerewolfNight)
	if err != nil {
		return PublicRoom{}, err
	}
	if !canActAtNight(player) {
		return PublicRoom{}, errors.New("role_has_no_night_action")
	}
	if actionID == "" {
		return PublicRoom{}, errors.New("invalid_target")
	}
	if !strings.Contains(actionID, ":") {
		actionID = "target:" + actionID
	}
	target, err := applyWerewolfNightAction(room, player, actionID)
	if err != nil {
		return PublicRoom{}, err
	}
	targetID := ""
	if target != nil {
		targetID = target.ID
	}
	recordAction(room, PublicAction{Type: "night_action", ActorID: player.ID, ActorName: player.Name, TargetID: targetID, Message: fmt.Sprintf("%s 完成了夜晚行动。", player.Name)})
	m.advanceWerewolfNight(room)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) HunterShot(roomID string, actorID string, targetID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfHunter {
		return PublicRoom{}, errors.New("invalid_phase")
	}
	hunter := findPlayerByID(room, room.Werewolf.HunterPendingID)
	player := findPlayerByUserID(room, actorID)
	if hunter == nil || player == nil || hunter.ID != player.ID || player.IsAI {
		return PublicRoom{}, errors.New("not_active_human_player")
	}
	if err := resolveHunterShot(room, targetID); err != nil {
		return PublicRoom{}, err
	}
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) AdvanceDay(roomID string, actorID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Game != GameWerewolf || room.Phase != PhaseWerewolfDay {
		return PublicRoom{}, errors.New("invalid_phase")
	}
	if room.HostUserID != actorID {
		return PublicRoom{}, errors.New("only_host_advance")
	}
	room.Phase = PhaseWerewolfVote
	room.Werewolf.Votes = map[string]string{}
	room.Log = append(room.Log, createLog("白天讨论结束，开始放逐投票。"))
	recordAction(room, PublicAction{Type: "vote_started", Message: "开始放逐投票。"})
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) WerewolfVote(roomID string, actorID string, targetID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireWerewolfActor(roomID, actorID, PhaseWerewolfVote)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Werewolf.RevealedIdiots[player.ID] {
		return PublicRoom{}, errors.New("idiot_cannot_vote_after_reveal")
	}
	target := findPlayerByID(room, targetID)
	if target == nil || !target.Alive {
		return PublicRoom{}, errors.New("invalid_target")
	}
	room.Werewolf.Votes[player.ID] = target.ID
	recordAction(room, PublicAction{Type: "vote", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 已投票。", player.Name)})
	m.resolveWerewolfVote(room)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) ProposeTeam(roomID string, actorID string, team []string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAvalonTeam)
	if err != nil {
		return PublicRoom{}, err
	}
	if room.Avalon.LeaderID != player.ID {
		return PublicRoom{}, errors.New("only_leader_propose")
	}
	if len(team) != room.Avalon.RequiredTeam {
		return PublicRoom{}, errors.New("invalid_team_size")
	}
	if hasDuplicate(team) {
		return PublicRoom{}, errors.New("duplicate_team_member")
	}
	for _, id := range team {
		target := findPlayerByID(room, id)
		if target == nil || !target.Alive {
			return PublicRoom{}, errors.New("invalid_team_member")
		}
	}

	room.Avalon.Team = append([]string{}, team...)
	room.Avalon.TeamVotes = map[string]bool{}
	room.Phase = PhaseAvalonVote
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 提名了任务队伍。", player.Name)))
	recordAction(room, PublicAction{Type: "team_proposed", ActorID: player.ID, ActorName: player.Name, Message: "任务队伍已提名。"})
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) TeamVote(roomID string, actorID string, approve bool) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAvalonVote)
	if err != nil {
		return PublicRoom{}, err
	}
	room.Avalon.TeamVotes[player.ID] = approve
	recordAction(room, PublicAction{Type: "team_vote", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 已投票。", player.Name)})
	m.resolveAvalonTeamVote(room)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) QuestCard(roomID string, actorID string, card string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAvalonQuest)
	if err != nil {
		return PublicRoom{}, err
	}
	if !slices.Contains(room.Avalon.Team, player.ID) {
		return PublicRoom{}, errors.New("not_on_quest_team")
	}
	card = strings.TrimSpace(card)
	if card != "success" && card != "fail" {
		return PublicRoom{}, errors.New("invalid_quest_card")
	}
	if card == "fail" && player.Alignment != AlignmentEvil {
		return PublicRoom{}, errors.New("good_player_cannot_fail")
	}
	room.Avalon.QuestCards[player.ID] = card
	recordAction(room, PublicAction{Type: "quest_card", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 已提交任务牌。", player.Name)})
	m.resolveAvalonQuest(room)
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) Assassinate(roomID string, actorID string, targetID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, player, err := m.requireAvalonActor(roomID, actorID, PhaseAssassination)
	if err != nil {
		return PublicRoom{}, err
	}
	if player.Role != RoleAssassin {
		return PublicRoom{}, errors.New("only_assassin")
	}
	target := findPlayerByID(room, targetID)
	if target == nil || target.Alignment != AlignmentGood {
		return PublicRoom{}, errors.New("invalid_target")
	}
	if target.Role == RoleMerlin {
		finish(room, AlignmentEvil, fmt.Sprintf("%s 刺中了梅林，邪恶阵营逆转获胜。", player.Name))
	} else {
		finish(room, AlignmentGood, fmt.Sprintf("%s 没有找到梅林，正义阵营获胜。", player.Name))
	}
	room.UpdatedAt = time.Now().UTC()
	return m.publicRoom(room, actorID), nil
}

func (m *Manager) RunNextAI(roomID string) (PublicRoom, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, false, err
	}
	if room.Phase == PhaseLobby || room.Phase == PhaseFinished {
		return m.publicRoom(room, ""), false, nil
	}

	acted := false
	for _, player := range room.Players {
		if !player.IsAI || !player.Alive {
			continue
		}
		if m.runAIAction(room, player) {
			acted = true
			break
		}
	}
	if acted {
		room.UpdatedAt = time.Now().UTC()
		return m.publicRoom(room, ""), room.Phase != PhaseFinished, nil
	}
	return m.publicRoom(room, ""), false, nil
}

func (m *Manager) Public(roomID string, viewerID string) (PublicRoom, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, err := m.room(roomID)
	if err != nil {
		return PublicRoom{}, err
	}
	return m.publicRoom(room, viewerID), nil
}

func (m *Manager) minPlayers() int {
	switch m.game {
	case GameWerewolf:
		return werewolfMinPlayers
	case GameUndercover:
		return undercoverMinPlayers
	default:
		return avalonMinPlayers
	}
}

func (m *Manager) maxPlayers() int {
	switch m.game {
	case GameWerewolf:
		return werewolfMaxPlayers
	case GameUndercover:
		return undercoverMaxPlayers
	default:
		return avalonMaxPlayers
	}
}

func (m *Manager) room(roomID string) (*Room, error) {
	roomID = strings.ToUpper(strings.TrimSpace(roomID))
	room := m.rooms[roomID]
	if room == nil {
		return nil, errors.New("room_not_found")
	}
	return room, nil
}

func (m *Manager) requireWerewolfActor(roomID string, actorID string, phase Phase) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Game != GameWerewolf || room.Phase != phase {
		return nil, nil, errors.New("invalid_phase")
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || !player.Alive || player.IsAI {
		return nil, nil, errors.New("not_active_human_player")
	}
	return room, player, nil
}

func (m *Manager) requireAvalonActor(roomID string, actorID string, phase Phase) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Game != GameAvalon || room.Phase != phase {
		return nil, nil, errors.New("invalid_phase")
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || player.IsAI {
		return nil, nil, errors.New("not_active_human_player")
	}
	return room, player, nil
}

func (m *Manager) requireUndercoverActor(roomID string, actorID string, phase Phase) (*Room, *Player, error) {
	room, err := m.room(roomID)
	if err != nil {
		return nil, nil, err
	}
	if room.Game != GameUndercover || room.Phase != phase {
		return nil, nil, errors.New("invalid_phase")
	}
	player := findPlayerByUserID(room, actorID)
	if player == nil || !player.Alive || player.IsAI {
		return nil, nil, errors.New("not_active_human_player")
	}
	return room, player, nil
}

func resetRoom(room *Room) {
	roleConfig := room.Werewolf.RoleConfig
	undercoverConfig := room.Undercover
	if undercoverConfig.PresetID == "" {
		undercoverConfig.PresetID = defaultUndercoverPresetID()
	}
	if room.Game == GameWerewolf && roleConfig.Counts.total() == 0 {
		roleConfig = defaultWerewolfConfig(len(room.Players))
	}
	room.Winner = ""
	room.WinnerMessage = ""
	room.RecentActions = nil
	room.Werewolf = WerewolfState{RoleConfig: roleConfig, RolePresets: werewolfRolePresets(len(room.Players)), NightActions: map[string]string{}, SeerChecks: map[string]Alignment{}, Votes: map[string]string{}, RevealedIdiots: map[string]bool{}, Day: 1}
	room.Avalon = AvalonState{TeamVotes: map[string]bool{}, QuestCards: map[string]string{}, Round: 1}
	room.Undercover = UndercoverState{PresetID: undercoverConfig.PresetID, IncludeBlank: undercoverConfig.IncludeBlank, Presets: undercoverPresets(), Described: map[string]bool{}, Votes: map[string]string{}, Round: 1}
	for index, player := range room.Players {
		player.Seat = index
		player.Alive = true
		player.Role = ""
		player.Alignment = ""
	}
}

func startWerewolf(room *Room) {
	if err := validateWerewolfCounts(room.Werewolf.RoleConfig.Counts, len(room.Players)); err != nil {
		applyDefaultWerewolfConfig(room)
	}
	roles := expandWerewolfRoles(room.Werewolf.RoleConfig.Counts)
	for index, player := range shuffledPlayers(room.Players) {
		player.Role = roles[index]
		player.Alignment = werewolfAlignment(player.Role)
	}
	room.Phase = PhaseWerewolfNight
	room.Werewolf.Day = 1
	room.Werewolf.RolePresets = nil
	room.Werewolf.NightActions = map[string]string{}
	room.Werewolf.SeerChecks = map[string]Alignment{}
	room.Werewolf.Votes = map[string]string{}
	room.Werewolf.RevealedIdiots = map[string]bool{}
	room.Werewolf.LastNight = ""
	room.Log = append(room.Log, createLog(fmt.Sprintf("狼人杀开始，角色组：%s。天黑请闭眼。", room.Werewolf.RoleConfig.Name)))
	recordAction(room, PublicAction{Type: "start", Message: "狼人杀开始，进入第一个夜晚。"})
}

func startAvalon(room *Room) {
	roles := avalonRoles(len(room.Players))
	for index, player := range shuffledPlayers(room.Players) {
		player.Role = roles[index]
		player.Alignment = avalonAlignment(player.Role)
	}
	room.Phase = PhaseAvalonTeam
	room.Avalon = AvalonState{
		Round:         1,
		LeaderID:      room.Players[0].ID,
		TeamVotes:     map[string]bool{},
		QuestCards:    map[string]string{},
		RequiredTeam:  avalonTeamSize(len(room.Players), 1),
		RequiredFails: avalonRequiredFails(len(room.Players), 1),
	}
	room.Log = append(room.Log, createLog("阿瓦隆开始，队长提名第一支任务队伍。"))
	recordAction(room, PublicAction{Type: "start", Message: "阿瓦隆开始，进入组队阶段。"})
}

func startUndercover(room *Room) {
	pair := chooseUndercoverPair(room.Undercover.PresetID)
	players := shuffledPlayers(room.Players)
	undercoverCount := undercoverCountForPlayers(len(players))
	blankCount := 0
	if room.Undercover.IncludeBlank && len(players) >= 6 {
		blankCount = 1
	}
	for index, player := range players {
		switch {
		case index < undercoverCount:
			player.Role = RoleUndercover
			player.Alignment = AlignmentEvil
		case index < undercoverCount+blankCount:
			player.Role = RoleBlank
			player.Alignment = AlignmentNeutral
		default:
			player.Role = RoleCivilian
			player.Alignment = AlignmentGood
		}
	}
	room.Phase = PhaseUndercoverDescribe
	room.Undercover.Round = 1
	room.Undercover.WordPair = pair
	room.Undercover.Presets = nil
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]string{}
	room.Undercover.CurrentSpeakerID = firstLivingPlayerID(room)
	room.Undercover.LastEliminatedID = ""
	room.Log = append(room.Log, createLog(fmt.Sprintf("谁是卧底开始，题库：%s。请依次描述自己的词。", undercoverPresetName(room.Undercover.PresetID))))
	recordAction(room, PublicAction{Type: "start", Message: "谁是卧底开始，进入描述阶段。"})
}

func werewolfAlignment(role Role) Alignment {
	if role == RoleWerewolf {
		return AlignmentEvil
	}
	return AlignmentGood
}

func advanceUndercoverSpeaker(room *Room) {
	next := nextUndescribedLivingPlayer(room)
	if next != nil {
		room.Undercover.CurrentSpeakerID = next.ID
		return
	}
	room.Phase = PhaseUndercoverVote
	room.Undercover.CurrentSpeakerID = ""
	room.Undercover.Votes = map[string]string{}
	room.Log = append(room.Log, createLog("本轮描述结束，开始投票。"))
	recordAction(room, PublicAction{Type: "undercover_vote_started", Message: "开始投票。"})
}

func resolveUndercoverVote(room *Room) {
	if len(room.Undercover.Votes) < livingCount(room) {
		return
	}
	targetID, tied := mostVotedUndercoverTarget(room.Undercover.Votes)
	if tied || targetID == "" {
		room.Log = append(room.Log, createLog("本轮投票平票，无人出局。"))
		recordAction(room, PublicAction{Type: "undercover_vote_tied", Message: "投票平票，无人出局。"})
		startNextUndercoverRound(room)
		return
	}
	target := findPlayerByID(room, targetID)
	if target != nil {
		target.Alive = false
		room.Undercover.LastEliminatedID = target.ID
		message := fmt.Sprintf("%s 被投票出局。", target.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "undercover_eliminate", TargetID: target.ID, Message: message})
	}
	if checkUndercoverWin(room) {
		return
	}
	startNextUndercoverRound(room)
}

func startNextUndercoverRound(room *Room) {
	room.Undercover.Round++
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]string{}
	room.Undercover.CurrentSpeakerID = firstLivingPlayerID(room)
	room.Phase = PhaseUndercoverDescribe
	room.Log = append(room.Log, createLog(fmt.Sprintf("第 %d 轮描述开始。", room.Undercover.Round)))
	recordAction(room, PublicAction{Type: "undercover_round_started", Message: "下一轮描述开始。"})
}

func checkUndercoverWin(room *Room) bool {
	civilians := 0
	undercover := 0
	blank := 0
	living := 0
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		living++
		switch player.Role {
		case RoleUndercover:
			undercover++
		case RoleBlank:
			blank++
		default:
			civilians++
		}
	}
	if undercover == 0 && blank == 0 {
		finish(room, AlignmentGood, "所有卧底阵营出局，平民获胜。")
		return true
	}
	if blank > 0 && undercover == 0 && living <= 2 {
		finish(room, AlignmentNeutral, "白板留到最后，白板获胜。")
		return true
	}
	if undercover+blank >= civilians || living <= 3 {
		finish(room, AlignmentEvil, "卧底阵营隐藏到最后，卧底获胜。")
		return true
	}
	return false
}

func applyDefaultWerewolfConfig(room *Room) {
	room.Werewolf.RoleConfig = defaultWerewolfConfig(len(room.Players))
	room.Werewolf.RolePresets = werewolfRolePresets(len(room.Players))
}

func reconcileWerewolfConfig(room *Room) {
	if room.Game != GameWerewolf || room.Phase != PhaseLobby {
		return
	}
	room.Werewolf.RolePresets = werewolfRolePresets(len(room.Players))
	if room.Werewolf.RoleConfig.Mode != "custom" {
		room.Werewolf.RoleConfig = defaultWerewolfConfig(len(room.Players))
		return
	}
	counts := room.Werewolf.RoleConfig.Counts
	counts.Villager += len(room.Players) - counts.total()
	if err := validateWerewolfCounts(counts, len(room.Players)); err != nil {
		room.Werewolf.RoleConfig = defaultWerewolfConfig(len(room.Players))
		return
	}
	room.Werewolf.RoleConfig.Counts = counts
	room.Werewolf.RoleConfig.Name = "自定义角色组"
}

func defaultWerewolfConfig(players int) WerewolfRoleConfig {
	presets := werewolfRolePresets(players)
	if len(presets) > 0 {
		preset := presets[0]
		return WerewolfRoleConfig{
			Mode:        "preset",
			PresetID:    preset.ID,
			Name:        preset.Name,
			Description: preset.Description,
			Counts:      preset.Counts,
		}
	}
	counts := fallbackWerewolfCounts(players)
	return WerewolfRoleConfig{
		Mode:        "custom",
		Name:        "自定义角色组",
		Description: "人数未满足开局时的临时角色配置。",
		Counts:      counts,
	}
}

func normalizeWerewolfConfig(config WerewolfRoleConfig, players int) (WerewolfRoleConfig, error) {
	mode := strings.TrimSpace(config.Mode)
	if mode == "" && strings.TrimSpace(config.PresetID) != "" {
		mode = "preset"
	}
	if mode != "custom" {
		for _, preset := range werewolfRolePresets(players) {
			if preset.ID == config.PresetID {
				return WerewolfRoleConfig{
					Mode:        "preset",
					PresetID:    preset.ID,
					Name:        preset.Name,
					Description: preset.Description,
					Counts:      preset.Counts,
				}, nil
			}
		}
		return WerewolfRoleConfig{}, errors.New("invalid_role_preset")
	}

	counts := config.Counts
	if err := validateWerewolfCounts(counts, players); err != nil {
		return WerewolfRoleConfig{}, err
	}
	return WerewolfRoleConfig{
		Mode:        "custom",
		Name:        "自定义角色组",
		Description: "由房主自由调整狼人、村民和神职数量。",
		Counts:      counts,
	}, nil
}

func werewolfRolePresets(players int) []WerewolfRolePreset {
	presets := map[int][]WerewolfRolePreset{
		6: {
			werewolfPreset("wwf-6-classic", "6人标准", "2 狼、1 预言家、1 女巫、2 村民。节奏直接，适合快速局。", 6, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Villager: 2}),
			werewolfPreset("wwf-6-guarded", "6人守卫局", "1 狼、1 预言家、1 女巫、1 守卫、2 村民。信息更安全。", 6, WerewolfRoleCounts{Werewolf: 1, Seer: 1, Witch: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-6-hunter", "6人猎人局", "2 狼、1 预言家、1 猎人、2 村民。死亡反击更刺激。", 6, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Hunter: 1, Villager: 2}),
		},
		7: {
			werewolfPreset("wwf-7-classic", "7人标准", "2 狼、1 预言家、1 女巫、1 猎人、2 村民。攻防均衡。", 7, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Villager: 2}),
			werewolfPreset("wwf-7-guarded", "7人守卫", "2 狼、1 预言家、1 女巫、1 守卫、2 村民。夜晚博弈更多。", 7, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-7-idiot", "7人白痴局", "2 狼、1 预言家、1 女巫、1 白痴、2 村民。放逐风险更高。", 7, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Idiot: 1, Villager: 2}),
		},
		8: {
			werewolfPreset("wwf-8-classic", "8人标准", "2 狼、预言家、女巫、猎人、白痴、2 村民。经典小板子。", 8, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 2}),
			werewolfPreset("wwf-8-guarded", "8人守卫", "2 狼、预言家、女巫、猎人、守卫、2 村民。夜晚更保守。", 8, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-8-wolfish", "8人狼压", "3 狼、预言家、女巫、猎人、2 村民。邪恶更强。", 8, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Villager: 2}),
		},
		9: {
			werewolfPreset("wwf-9-classic", "9人预女猎白", "3 狼、预言家、女巫、猎人、白痴、2 村民。发言强度高。", 9, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 2}),
			werewolfPreset("wwf-9-guarded", "9人守卫", "3 狼、预言家、女巫、猎人、守卫、2 村民。攻防均衡。", 9, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 2}),
			werewolfPreset("wwf-9-balanced", "9人均衡", "2 狼、预言家、女巫、猎人、白痴、3 村民。好人容错更高。", 9, WerewolfRoleCounts{Werewolf: 2, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 3}),
		},
		10: {
			werewolfPreset("wwf-10-classic", "10人预女猎白", "3 狼、预言家、女巫、猎人、白痴、3 村民。推荐配置。", 10, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 3}),
			werewolfPreset("wwf-10-guarded", "10人守卫", "3 狼、预言家、女巫、猎人、守卫、3 村民。防守变量更多。", 10, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 3}),
			werewolfPreset("wwf-10-fullgod", "10人五神", "3 狼、预言家、女巫、猎人、白痴、守卫、2 村民。神职密度高。", 10, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 2}),
		},
		11: {
			werewolfPreset("wwf-11-classic", "11人经典", "3 狼、预言家、女巫、猎人、白痴、守卫、3 村民。信息与狼刀平衡。", 11, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 3}),
			werewolfPreset("wwf-11-wolfish", "11人狼压", "4 狼、预言家、女巫、猎人、白痴、3 村民。高压对抗。", 11, WerewolfRoleCounts{Werewolf: 4, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 3}),
			werewolfPreset("wwf-11-safe", "11人稳健", "3 狼、预言家、女巫、猎人、守卫、4 村民。适合轻松局。", 11, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Guard: 1, Villager: 4}),
		},
		12: {
			werewolfPreset("wwf-12-classic", "12人经典", "4 狼、预言家、女巫、猎人、白痴、守卫、3 村民。满桌推荐。", 12, WerewolfRoleCounts{Werewolf: 4, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 3}),
			werewolfPreset("wwf-12-balanced", "12人均衡", "3 狼、预言家、女巫、猎人、白痴、守卫、4 村民。讨论时间更宽。", 12, WerewolfRoleCounts{Werewolf: 3, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Guard: 1, Villager: 4}),
			werewolfPreset("wwf-12-wolfish", "12人狼压", "4 狼、预言家、女巫、猎人、白痴、4 村民。狼队进攻更直接。", 12, WerewolfRoleCounts{Werewolf: 4, Seer: 1, Witch: 1, Hunter: 1, Idiot: 1, Villager: 4}),
		},
	}
	return append([]WerewolfRolePreset{}, presets[players]...)
}

func werewolfPreset(id string, name string, description string, players int, counts WerewolfRoleCounts) WerewolfRolePreset {
	return WerewolfRolePreset{ID: id, Name: name, Description: description, Players: players, Counts: counts}
}

func fallbackWerewolfCounts(players int) WerewolfRoleCounts {
	if players <= 0 {
		return WerewolfRoleCounts{}
	}
	counts := WerewolfRoleCounts{Villager: players}
	if players >= 2 {
		counts.Werewolf = 1
		counts.Villager--
	}
	if players >= 3 {
		counts.Seer = 1
		counts.Villager--
	}
	if players >= 4 {
		counts.Guard = 1
		counts.Villager--
	}
	if players >= 6 {
		counts.Witch = 1
		counts.Villager--
	}
	return counts
}

func expandWerewolfRoles(counts WerewolfRoleCounts) []Role {
	roles := []Role{}
	for range counts.Werewolf {
		roles = append(roles, RoleWerewolf)
	}
	for range counts.Seer {
		roles = append(roles, RoleSeer)
	}
	for range counts.Guard {
		roles = append(roles, RoleGuard)
	}
	for range counts.Witch {
		roles = append(roles, RoleWitch)
	}
	for range counts.Hunter {
		roles = append(roles, RoleHunter)
	}
	for range counts.Idiot {
		roles = append(roles, RoleIdiot)
	}
	for range counts.Villager {
		roles = append(roles, RoleVillager)
	}
	return roles
}

func validateWerewolfCounts(counts WerewolfRoleCounts, players int) error {
	if counts.Villager < 0 || counts.Werewolf < 0 || counts.Seer < 0 || counts.Guard < 0 || counts.Witch < 0 || counts.Hunter < 0 || counts.Idiot < 0 {
		return errors.New("invalid_role_count")
	}
	if counts.total() != players {
		return errors.New("role_count_mismatch")
	}
	if players >= werewolfMinPlayers && counts.Werewolf < 1 {
		return errors.New("need_werewolf")
	}
	if counts.Werewolf >= players {
		return errors.New("too_many_werewolves")
	}
	if counts.Seer > 1 || counts.Guard > 1 || counts.Witch > 1 || counts.Hunter > 1 || counts.Idiot > 1 {
		return errors.New("too_many_unique_roles")
	}
	return nil
}

func (counts WerewolfRoleCounts) total() int {
	return counts.Villager + counts.Werewolf + counts.Seer + counts.Guard + counts.Witch + counts.Hunter + counts.Idiot
}

func avalonRoles(count int) []Role {
	evil := map[int]int{5: 2, 6: 2, 7: 3, 8: 3, 9: 3, 10: 4}[count]
	roles := []Role{RoleMerlin, RoleAssassin}
	for len(roles) < evil+1 {
		roles = append(roles, RoleMinion)
	}
	for len(roles) < count {
		roles = append(roles, RoleLoyal)
	}
	return roles
}

func avalonAlignment(role Role) Alignment {
	if role == RoleAssassin || role == RoleMinion {
		return AlignmentEvil
	}
	return AlignmentGood
}

func (m *Manager) advanceWerewolfNight(room *Room) {
	if !allRequiredNightActions(room) {
		return
	}

	killID := mostVotedTarget(room.Werewolf.NightActions, func(playerID string) bool {
		player := findPlayerByID(room, playerID)
		return player != nil && player.Role == RoleWerewolf
	})
	protectedID := ""
	for playerID, targetID := range room.Werewolf.NightActions {
		player := findPlayerByID(room, playerID)
		if player != nil && player.Role == RoleGuard {
			protectedID = targetID
		}
	}
	if room.Werewolf.WitchSaveTargetID != "" && room.Werewolf.WitchSaveTargetID == killID {
		protectedID = killID
	}

	deaths := []*Player{}
	if killID != "" && killID != protectedID {
		if target := findPlayerByID(room, killID); target != nil && target.Alive {
			deaths = append(deaths, target)
		}
	}
	if room.Werewolf.WitchPoisonID != "" {
		if target := findPlayerByID(room, room.Werewolf.WitchPoisonID); target != nil && target.Alive && !slices.Contains(deaths, target) {
			deaths = append(deaths, target)
		}
	}

	if len(deaths) == 0 {
		room.Werewolf.LastNight = "昨夜无人出局。"
		room.Log = append(room.Log, createLog(room.Werewolf.LastNight))
	} else {
		names := []string{}
		for _, target := range deaths {
			target.Alive = false
			names = append(names, target.Name)
		}
		room.Werewolf.LastNight = fmt.Sprintf("%s 在夜晚出局。", strings.Join(names, "、"))
		room.Log = append(room.Log, createLog(room.Werewolf.LastNight))
	}
	room.Werewolf.WitchSaveTargetID = ""
	room.Werewolf.WitchPoisonID = ""

	if hunter := firstDeadHunter(deaths); hunter != nil {
		room.Phase = PhaseWerewolfHunter
		room.Werewolf.HunterPendingID = hunter.ID
		room.Werewolf.HunterAfterPhase = PhaseWerewolfDay
		recordAction(room, PublicAction{Type: "hunter_pending", ActorID: hunter.ID, ActorName: hunter.Name, Message: fmt.Sprintf("%s 可以发动猎人技能。", hunter.Name)})
		return
	}

	if checkWerewolfWin(room) {
		return
	}
	room.Phase = PhaseWerewolfDay
	room.Werewolf.NightActions = map[string]string{}
	room.Werewolf.Votes = map[string]string{}
	recordAction(room, PublicAction{Type: "day_started", Message: room.Werewolf.LastNight})
}

func allRequiredNightActions(room *Room) bool {
	werewolfActed := false
	werewolfAlive := false
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		switch player.Role {
		case RoleWerewolf:
			werewolfAlive = true
			if _, ok := room.Werewolf.NightActions[player.ID]; ok {
				werewolfActed = true
			}
		case RoleSeer, RoleGuard:
			if _, ok := room.Werewolf.NightActions[player.ID]; !ok {
				return false
			}
		case RoleWitch:
			if witchCanAct(room) {
				if _, ok := room.Werewolf.NightActions[player.ID]; !ok {
					return false
				}
			}
		}
	}
	return !werewolfAlive || werewolfActed
}

func canActAtNight(player *Player) bool {
	return player.Role == RoleWerewolf || player.Role == RoleSeer || player.Role == RoleGuard || player.Role == RoleWitch
}

func witchCanAct(room *Room) bool {
	return !room.Werewolf.WitchAntidoteUsed || !room.Werewolf.WitchPoisonUsed
}

func applyWerewolfNightAction(room *Room, player *Player, actionID string) (*Player, error) {
	actions := werewolfNightActions(room, player)
	if !aiplayer.ValidateAction(actionID, actions) {
		return nil, errors.New("invalid_target")
	}
	switch {
	case strings.HasPrefix(actionID, "skip:"):
		room.Werewolf.NightActions[player.ID] = actionID
		return nil, nil
	case strings.HasPrefix(actionID, "save:"):
		target := playerFromAction(room, actionID, "save:")
		if target == nil || !target.Alive || room.Werewolf.WitchAntidoteUsed {
			return nil, errors.New("invalid_target")
		}
		room.Werewolf.WitchAntidoteUsed = true
		room.Werewolf.WitchSaveTargetID = target.ID
		room.Werewolf.NightActions[player.ID] = actionID
		return target, nil
	case strings.HasPrefix(actionID, "poison:"):
		target := playerFromAction(room, actionID, "poison:")
		if target == nil || !target.Alive || room.Werewolf.WitchPoisonUsed {
			return nil, errors.New("invalid_target")
		}
		room.Werewolf.WitchPoisonUsed = true
		room.Werewolf.WitchPoisonID = target.ID
		room.Werewolf.NightActions[player.ID] = actionID
		return target, nil
	default:
		target := playerFromAction(room, actionID, "target:")
		if target == nil || !target.Alive {
			return nil, errors.New("invalid_target")
		}
		room.Werewolf.NightActions[player.ID] = target.ID
		if player.Role == RoleSeer {
			room.Werewolf.SeerChecks[target.ID] = target.Alignment
		}
		return target, nil
	}
}

func resolveHunterShot(room *Room, targetID string) error {
	hunter := findPlayerByID(room, room.Werewolf.HunterPendingID)
	if hunter == nil || hunter.Role != RoleHunter {
		return errors.New("hunter_not_found")
	}
	targetID = strings.TrimSpace(targetID)
	if targetID != "" {
		target := findPlayerByID(room, targetID)
		if target == nil || !target.Alive || target.ID == hunter.ID {
			return errors.New("invalid_target")
		}
		target.Alive = false
		message := fmt.Sprintf("%s 发动猎人技能带走了 %s。", hunter.Name, target.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "hunter_shot", ActorID: hunter.ID, ActorName: hunter.Name, TargetID: target.ID, Message: message})
	} else {
		message := fmt.Sprintf("%s 放弃发动猎人技能。", hunter.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "hunter_skip", ActorID: hunter.ID, ActorName: hunter.Name, Message: message})
	}
	afterPhase := room.Werewolf.HunterAfterPhase
	room.Werewolf.HunterPendingID = ""
	room.Werewolf.HunterAfterPhase = ""
	if checkWerewolfWin(room) {
		return nil
	}
	if afterPhase == PhaseWerewolfDay {
		room.Phase = PhaseWerewolfDay
		room.Werewolf.NightActions = map[string]string{}
		room.Werewolf.Votes = map[string]string{}
		recordAction(room, PublicAction{Type: "day_started", Message: room.Werewolf.LastNight})
		return nil
	}
	startNextWerewolfNight(room)
	return nil
}

func firstDeadHunter(players []*Player) *Player {
	for _, player := range players {
		if player.Role == RoleHunter {
			return player
		}
	}
	return nil
}

func werewolfVoterCount(room *Room) int {
	count := 0
	for _, player := range room.Players {
		if player.Alive && !room.Werewolf.RevealedIdiots[player.ID] {
			count++
		}
	}
	return count
}

func (m *Manager) resolveWerewolfVote(room *Room) {
	if len(room.Werewolf.Votes) < werewolfVoterCount(room) {
		return
	}
	targetID := mostVotedTarget(room.Werewolf.Votes, func(string) bool { return true })
	if target := findPlayerByID(room, targetID); target != nil {
		if target.Role == RoleIdiot && !room.Werewolf.RevealedIdiots[target.ID] {
			if room.Werewolf.RevealedIdiots == nil {
				room.Werewolf.RevealedIdiots = map[string]bool{}
			}
			room.Werewolf.RevealedIdiots[target.ID] = true
			message := fmt.Sprintf("%s 是白痴，翻牌免疫本次放逐。", target.Name)
			room.Log = append(room.Log, createLog(message))
			recordAction(room, PublicAction{Type: "idiot_revealed", TargetID: target.ID, Message: message})
			startNextWerewolfNight(room)
			return
		}
		target.Alive = false
		message := fmt.Sprintf("%s 被放逐出局。", target.Name)
		room.Log = append(room.Log, createLog(message))
		recordAction(room, PublicAction{Type: "exile", TargetID: target.ID, Message: message})
		if target.Role == RoleHunter {
			room.Phase = PhaseWerewolfHunter
			room.Werewolf.HunterPendingID = target.ID
			room.Werewolf.HunterAfterPhase = PhaseWerewolfNight
			recordAction(room, PublicAction{Type: "hunter_pending", ActorID: target.ID, ActorName: target.Name, Message: fmt.Sprintf("%s 可以发动猎人技能。", target.Name)})
			return
		}
	}
	if checkWerewolfWin(room) {
		return
	}
	startNextWerewolfNight(room)
}

func startNextWerewolfNight(room *Room) {
	room.Werewolf.Day++
	room.Werewolf.Votes = map[string]string{}
	room.Werewolf.NightActions = map[string]string{}
	room.Phase = PhaseWerewolfNight
	room.Log = append(room.Log, createLog("夜幕再次降临。"))
	recordAction(room, PublicAction{Type: "night_started", Message: "夜幕再次降临。"})
}

func checkWerewolfWin(room *Room) bool {
	wolves := 0
	good := 0
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		if player.Alignment == AlignmentEvil {
			wolves++
		} else {
			good++
		}
	}
	if wolves == 0 {
		finish(room, AlignmentGood, "所有狼人出局，好人阵营获胜。")
		return true
	}
	if wolves >= good {
		finish(room, AlignmentEvil, "狼人数量已压制好人，狼人阵营获胜。")
		return true
	}
	return false
}

func (m *Manager) resolveAvalonTeamVote(room *Room) {
	if len(room.Avalon.TeamVotes) < len(room.Players) {
		return
	}
	approve := 0
	for _, vote := range room.Avalon.TeamVotes {
		if vote {
			approve++
		}
	}
	if approve > len(room.Players)/2 {
		room.Phase = PhaseAvalonQuest
		room.Avalon.QuestCards = map[string]string{}
		room.Log = append(room.Log, createLog("任务队伍通过，队员开始提交任务牌。"))
		recordAction(room, PublicAction{Type: "team_approved", Message: "任务队伍通过。"})
		return
	}

	room.Avalon.RejectedTeams++
	if room.Avalon.RejectedTeams >= 5 {
		finish(room, AlignmentEvil, "连续五次组队失败，邪恶阵营获胜。")
		return
	}
	room.Phase = PhaseAvalonTeam
	room.Avalon.Team = nil
	room.Avalon.TeamVotes = map[string]bool{}
	advanceAvalonLeader(room)
	room.Log = append(room.Log, createLog("任务队伍未通过，下一位队长重新提名。"))
	recordAction(room, PublicAction{Type: "team_rejected", Message: "任务队伍未通过。"})
}

func (m *Manager) resolveAvalonQuest(room *Room) {
	if len(room.Avalon.QuestCards) < len(room.Avalon.Team) {
		return
	}
	failCards := 0
	for _, card := range room.Avalon.QuestCards {
		if card == "fail" {
			failCards++
		}
	}
	result := AvalonQuestResult{Round: room.Avalon.Round, TeamSize: len(room.Avalon.Team), FailCards: failCards}
	room.Avalon.QuestResults = append(room.Avalon.QuestResults, result)
	if failCards >= room.Avalon.RequiredFails {
		room.Avalon.Fails++
		room.Log = append(room.Log, createLog(fmt.Sprintf("第 %d 次任务失败。", room.Avalon.Round)))
	} else {
		room.Avalon.Successes++
		room.Log = append(room.Log, createLog(fmt.Sprintf("第 %d 次任务成功。", room.Avalon.Round)))
	}

	if room.Avalon.Fails >= 3 {
		finish(room, AlignmentEvil, "三次任务失败，邪恶阵营获胜。")
		return
	}
	if room.Avalon.Successes >= 3 {
		room.Phase = PhaseAssassination
		room.Log = append(room.Log, createLog("正义阵营完成三次任务，刺客最后寻找梅林。"))
		recordAction(room, PublicAction{Type: "assassination_started", Message: "刺杀阶段开始。"})
		return
	}

	room.Avalon.Round++
	room.Avalon.Team = nil
	room.Avalon.TeamVotes = map[string]bool{}
	room.Avalon.QuestCards = map[string]string{}
	room.Avalon.RequiredTeam = avalonTeamSize(len(room.Players), room.Avalon.Round)
	room.Avalon.RequiredFails = avalonRequiredFails(len(room.Players), room.Avalon.Round)
	room.Phase = PhaseAvalonTeam
	advanceAvalonLeader(room)
	recordAction(room, PublicAction{Type: "quest_resolved", Message: "任务结算完成。"})
}

func (m *Manager) runAIAction(room *Room, player *Player) bool {
	switch room.Phase {
	case PhaseWerewolfNight:
		if !canActAtNight(player) {
			return false
		}
		if _, ok := room.Werewolf.NightActions[player.ID]; ok {
			return false
		}
		actionID, speech := m.chooseWerewolfNightAction(room, player)
		if actionID == "" {
			return false
		}
		if _, err := applyWerewolfNightAction(room, player, actionID); err != nil {
			return false
		}
		if speech == "" {
			speech = "今晚先看这一位。"
		}
		recordSpeech(room, player, speech)
		m.advanceWerewolfNight(room)
		return true
	case PhaseWerewolfVote:
		if _, ok := room.Werewolf.Votes[player.ID]; ok {
			return false
		}
		if room.Werewolf.RevealedIdiots[player.ID] {
			return false
		}
		target, speech := m.chooseWerewolfVote(room, player)
		if target == nil {
			return false
		}
		room.Werewolf.Votes[player.ID] = target.ID
		if speech == "" {
			speech = "我投这里。"
		}
		recordSpeech(room, player, speech)
		m.resolveWerewolfVote(room)
		return true
	case PhaseWerewolfHunter:
		if room.Werewolf.HunterPendingID != player.ID {
			return false
		}
		target, speech, ok := m.chooseHunterShot(room, player)
		if !ok {
			return false
		}
		if speech != "" {
			recordSpeech(room, player, speech)
		}
		targetID := ""
		if target != nil {
			targetID = target.ID
		}
		return resolveHunterShot(room, targetID) == nil
	case PhaseAvalonTeam:
		if room.Avalon.LeaderID != player.ID {
			return false
		}
		team, speech := m.chooseAvalonTeam(room, player)
		if len(team) != room.Avalon.RequiredTeam {
			return false
		}
		room.Avalon.Team = team
		room.Avalon.TeamVotes = map[string]bool{}
		room.Phase = PhaseAvalonVote
		if speech == "" {
			speech = "这队先试一下。"
		}
		recordSpeech(room, player, speech)
		return true
	case PhaseAvalonVote:
		if _, ok := room.Avalon.TeamVotes[player.ID]; ok {
			return false
		}
		approve, speech, ok := m.chooseAvalonTeamVote(room, player)
		if !ok {
			return false
		}
		room.Avalon.TeamVotes[player.ID] = approve
		if speech == "" {
			speech = "我给过。"
		}
		recordSpeech(room, player, speech)
		m.resolveAvalonTeamVote(room)
		return true
	case PhaseAvalonQuest:
		if !slices.Contains(room.Avalon.Team, player.ID) {
			return false
		}
		if _, ok := room.Avalon.QuestCards[player.ID]; ok {
			return false
		}
		card, speech := m.chooseAvalonQuestCard(room, player)
		if card == "" {
			return false
		}
		room.Avalon.QuestCards[player.ID] = card
		if speech != "" {
			recordSpeech(room, player, speech)
		}
		m.resolveAvalonQuest(room)
		return true
	case PhaseAssassination:
		if player.Role != RoleAssassin {
			return false
		}
		target, speech := m.chooseAvalonAssassination(room, player)
		if target == nil {
			return false
		}
		if speech != "" {
			recordSpeech(room, player, speech)
		}
		if target.Role == RoleMerlin {
			finish(room, AlignmentEvil, fmt.Sprintf("%s 刺中了梅林，邪恶阵营逆转获胜。", player.Name))
		} else {
			finish(room, AlignmentGood, fmt.Sprintf("%s 没有找到梅林，正义阵营获胜。", player.Name))
		}
		return true
	case PhaseUndercoverDescribe:
		if room.Undercover.CurrentSpeakerID != player.ID {
			return false
		}
		text := m.chooseUndercoverDescription(room, player)
		if text == "" {
			return false
		}
		recordSpeech(room, player, text)
		room.Undercover.Described[player.ID] = true
		recordAction(room, PublicAction{Type: "undercover_describe", ActorID: player.ID, ActorName: player.Name, Message: fmt.Sprintf("%s 完成了描述。", player.Name)})
		advanceUndercoverSpeaker(room)
		return true
	case PhaseUndercoverVote:
		if _, ok := room.Undercover.Votes[player.ID]; ok {
			return false
		}
		target := m.chooseUndercoverVote(room, player)
		if target == nil {
			return false
		}
		room.Undercover.Votes[player.ID] = target.ID
		recordSpeech(room, player, "我先票这个位置。")
		recordAction(room, PublicAction{Type: "undercover_vote", ActorID: player.ID, ActorName: player.Name, TargetID: target.ID, Message: fmt.Sprintf("%s 已投票。", player.Name)})
		resolveUndercoverVote(room)
		return true
	default:
		return false
	}
}

func finish(room *Room, winner Alignment, message string) {
	room.Phase = PhaseFinished
	room.Winner = winner
	room.WinnerMessage = message
	room.Log = append(room.Log, createLog(message))
	recordAction(room, PublicAction{Type: "finished", Message: message})
}

func (m *Manager) publicRoom(room *Room, viewerUserID string) PublicRoom {
	viewer := findPlayerByUserID(room, viewerUserID)
	players := make([]PublicPlayer, 0, len(room.Players))
	for _, player := range room.Players {
		visible := roleVisible(room, viewer, player)
		publicPlayer := PublicPlayer{
			ID:             player.ID,
			UserID:         player.UserID,
			Name:           player.Name,
			Seat:           player.Seat,
			RoomRole:       player.RoomRole,
			Kind:           player.Kind,
			IsAI:           player.IsAI,
			Connected:      player.Connected,
			DisconnectedAt: player.DisconnectedAt,
			AI:             player.AI,
			Alive:          player.Alive,
			VisibleToYou:   visible,
			Note:           playerNote(room, viewer, player.ID),
		}
		if visible {
			publicPlayer.Role = player.Role
			publicPlayer.Alignment = player.Alignment
		}
		players = append(players, publicPlayer)
	}

	logs := room.Log
	if len(logs) > 12 {
		logs = logs[len(logs)-12:]
	}
	youPlayerID := ""
	if viewer != nil {
		youPlayerID = viewer.ID
	}

	return PublicRoom{
		ID:            room.ID,
		Game:          room.Game,
		HostUserID:    room.HostUserID,
		Phase:         room.Phase,
		Players:       players,
		YouPlayerID:   youPlayerID,
		MinPlayers:    m.minPlayers(),
		MaxPlayers:    m.maxPlayers(),
		Werewolf:      werewolfViewForViewer(room, viewer),
		Avalon:        AvalonView{Round: room.Avalon.Round, LeaderID: room.Avalon.LeaderID, Team: append([]string{}, room.Avalon.Team...), TeamVotes: cloneBoolMap(room.Avalon.TeamVotes), QuestResults: append([]AvalonQuestResult{}, room.Avalon.QuestResults...), RejectedTeams: room.Avalon.RejectedTeams, RequiredTeam: room.Avalon.RequiredTeam, RequiredFails: room.Avalon.RequiredFails, Successes: room.Avalon.Successes, Fails: room.Avalon.Fails},
		Undercover:    undercoverViewForViewer(room, viewer),
		Winner:        room.Winner,
		WinnerMessage: room.WinnerMessage,
		Log:           append([]LogEntry{}, logs...),
		Speeches:      append([]SpeechEntry{}, room.Speeches...),
		ActionSeq:     room.ActionSeq,
		RecentActions: append([]PublicAction{}, room.RecentActions...),
	}
}

func roleVisible(room *Room, viewer *Player, target *Player) bool {
	if room.Phase == PhaseFinished {
		return true
	}
	if viewer == nil || target.Role == "" {
		return false
	}
	if viewer.ID == target.ID {
		return true
	}
	if room.Game == GameWerewolf {
		return viewer.Role == RoleWerewolf && target.Role == RoleWerewolf
	}
	if room.Game == GameAvalon {
		if viewer.Alignment == AlignmentEvil && target.Alignment == AlignmentEvil {
			return true
		}
		return viewer.Role == RoleMerlin && target.Alignment == AlignmentEvil
	}
	if room.Game == GameUndercover {
		return viewer.ID == target.ID
	}
	return false
}

func werewolfViewForViewer(room *Room, viewer *Player) WerewolfView {
	view := WerewolfView{
		Day:             room.Werewolf.Day,
		RoleConfig:      room.Werewolf.RoleConfig,
		RolePresets:     werewolfRolePresets(len(room.Players)),
		SeerChecks:      seerChecksForViewer(room, viewer),
		Votes:           cloneStringMap(room.Werewolf.Votes),
		LastNight:       room.Werewolf.LastNight,
		HunterPendingID: room.Werewolf.HunterPendingID,
		RevealedIdiots:  cloneBoolMap(room.Werewolf.RevealedIdiots),
	}
	if viewer != nil && viewer.Role == RoleWitch {
		view.WitchVictimID = currentWerewolfKillTarget(room)
		view.WitchAntidoteUsed = room.Werewolf.WitchAntidoteUsed
		view.WitchPoisonUsed = room.Werewolf.WitchPoisonUsed
	}
	return view
}

func avalonTeamSize(players int, round int) int {
	table := map[int][]int{
		5:  {2, 3, 2, 3, 3},
		6:  {2, 3, 4, 3, 4},
		7:  {2, 3, 3, 4, 4},
		8:  {3, 4, 4, 5, 5},
		9:  {3, 4, 4, 5, 5},
		10: {3, 4, 4, 5, 5},
	}
	return table[players][round-1]
}

func avalonRequiredFails(players int, round int) int {
	if players >= 7 && round == 4 {
		return 2
	}
	return 1
}

func advanceAvalonLeader(room *Room) {
	index := 0
	for currentIndex, player := range room.Players {
		if player.ID == room.Avalon.LeaderID {
			index = currentIndex
			break
		}
	}
	room.Avalon.LeaderID = room.Players[(index+1)%len(room.Players)].ID
}

func (m *Manager) chooseWerewolfNightAction(room *Room, actor *Player) (string, string) {
	actions := werewolfNightActions(room, actor)
	if len(actions) == 0 {
		return "", ""
	}
	if m.canUseLLM(actor) {
		decision, err := m.socialDecision(room, actor, werewolfAIState(room, actor), actions)
		if err == nil && aiplayer.ValidateAction(decision.ActionID, actions) {
			return decision.ActionID, strings.TrimSpace(decision.Speech)
		}
		if err != nil {
			slog.Warn("werewolf llm night action failed", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "error", err)
		} else {
			slog.Warn("werewolf llm night action invalid", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "actionID", decision.ActionID)
		}
	}
	return "", ""
}

func (m *Manager) chooseWerewolfVote(room *Room, actor *Player) (*Player, string) {
	actions := werewolfVoteActions(room, actor)
	if len(actions) == 0 {
		return nil, ""
	}
	if m.canUseLLM(actor) {
		decision, err := m.socialDecision(room, actor, werewolfAIState(room, actor), actions)
		if err == nil && aiplayer.ValidateAction(decision.ActionID, actions) {
			if target := playerFromAction(room, decision.ActionID, "vote:"); target != nil && target.Alive {
				return target, strings.TrimSpace(decision.Speech)
			}
		}
		if err != nil {
			slog.Warn("werewolf llm vote failed", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "error", err)
		} else {
			slog.Warn("werewolf llm vote invalid", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "actionID", decision.ActionID)
		}
	}
	return nil, ""
}

func (m *Manager) chooseHunterShot(room *Room, actor *Player) (*Player, string, bool) {
	actions := hunterShotActions(room, actor)
	if len(actions) == 0 {
		return nil, "", false
	}
	if m.canUseLLM(actor) {
		decision, err := m.socialDecision(room, actor, werewolfAIState(room, actor), actions)
		if err == nil && aiplayer.ValidateAction(decision.ActionID, actions) {
			if decision.ActionID == "shoot:skip" {
				return nil, strings.TrimSpace(decision.Speech), true
			}
			target := playerFromAction(room, decision.ActionID, "shoot:")
			if target != nil && target.Alive {
				return target, strings.TrimSpace(decision.Speech), true
			}
		}
		if err != nil {
			slog.Warn("werewolf llm hunter shot failed", "room", room.ID, "player", actor.ID, "playerName", actor.Name, "error", err)
		}
	}
	return nil, "", false
}

func (m *Manager) canUseLLM(player *Player) bool {
	return player.AI != nil && player.AI.Level == string(aiplayer.LevelLLM) && m.aiProvider != nil && m.aiProvider.Enabled()
}

type socialAISession struct {
	Game        GameKind
	RoomID      string
	PlayerID    string
	PrivateRole Role
	PrivateWord string
	Memory      []string
}

func (m *Manager) ensureAISession(room *Room, player *Player) *socialAISession {
	key := socialAISessionKey(room.ID, player.ID)
	session := m.aiSessions[key]
	if session == nil {
		session = &socialAISession{Game: room.Game, RoomID: room.ID, PlayerID: player.ID}
		m.aiSessions[key] = session
	}
	session.PrivateRole = player.Role
	session.PrivateWord = undercoverWordForPlayer(room, player)
	return session
}

func (m *Manager) socialDecision(room *Room, player *Player, state map[string]any, actions []aiplayer.LegalAction) (aiplayer.Decision, error) {
	if !m.canUseLLM(player) {
		return aiplayer.Decision{}, errors.New("llm_not_configured")
	}
	session := m.ensureAISession(room, player)
	state["aiSession"] = map[string]any{
		"sessionId": sessionID(room, player),
		"memory":    append([]string{}, session.Memory...),
	}
	state["privateNotes"] = cloneStringMap(room.PlayerNotes[player.ID])
	decision, err := m.aiProvider.Decide(context.Background(), aiplayer.DecisionInput{
		Game:        string(room.Game),
		Level:       aiplayer.LevelLLM,
		SessionID:   sessionID(room, player),
		PlayerName:  player.Name,
		Personality: player.AI.Personality,
		SpeechStyle: player.AI.SpeechStyle,
		State:       state,
		Actions:     actions,
	})
	if err != nil {
		return decision, err
	}
	m.applyAINotes(room, player, decision.Notes)
	m.rememberAI(room, player, fmt.Sprintf("phase=%s action=%s reason=%s speech=%s", room.Phase, decision.ActionID, strings.TrimSpace(decision.Reason), strings.TrimSpace(decision.Speech)))
	return decision, nil
}

func (m *Manager) applyAINotes(room *Room, player *Player, notes map[string]string) {
	for targetID, note := range notes {
		if findPlayerByID(room, targetID) == nil {
			continue
		}
		setPlayerNote(room, player.ID, targetID, note)
	}
}

func (m *Manager) rememberAI(room *Room, player *Player, event string) {
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
	return fmt.Sprintf("social:%s:%s:%s", room.Game, room.ID, player.ID)
}

func socialAISessionKey(roomID string, playerID string) string {
	return roomID + ":" + playerID
}

func werewolfNightActions(room *Room, actor *Player) []aiplayer.LegalAction {
	if !canActAtNight(actor) {
		return nil
	}
	if actor.Role == RoleWitch {
		return witchNightActions(room, actor)
	}
	labelPrefix := "选择"
	switch actor.Role {
	case RoleWerewolf:
		labelPrefix = "击杀"
	case RoleSeer:
		labelPrefix = "查验"
	case RoleGuard:
		labelPrefix = "守护"
	}
	actions := []aiplayer.LegalAction{}
	for _, target := range room.Players {
		if !target.Alive {
			continue
		}
		if actor.Role == RoleWerewolf && target.Role == RoleWerewolf {
			continue
		}
		if actor.Role == RoleSeer && target.ID == actor.ID {
			continue
		}
		actions = append(actions, aiplayer.LegalAction{
			ID:          "target:" + target.ID,
			Label:       fmt.Sprintf("%s %s", labelPrefix, target.Name),
			Description: fmt.Sprintf("座位 %d 的存活玩家", target.Seat+1),
		})
	}
	return actions
}

func witchNightActions(room *Room, actor *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	killID := currentWerewolfKillTarget(room)
	if killID != "" && !room.Werewolf.WitchAntidoteUsed {
		if target := findPlayerByID(room, killID); target != nil && target.Alive {
			actions = append(actions, aiplayer.LegalAction{
				ID:          "save:" + target.ID,
				Label:       fmt.Sprintf("使用解药救 %s", target.Name),
				Description: "消耗一次解药，阻止今晚狼刀出局。",
			})
		}
	}
	if !room.Werewolf.WitchPoisonUsed {
		for _, target := range room.Players {
			if !target.Alive || target.ID == actor.ID {
				continue
			}
			actions = append(actions, aiplayer.LegalAction{
				ID:          "poison:" + target.ID,
				Label:       fmt.Sprintf("使用毒药毒 %s", target.Name),
				Description: fmt.Sprintf("消耗一次毒药，令座位 %d 出局。", target.Seat+1),
			})
		}
	}
	actions = append(actions, aiplayer.LegalAction{ID: "skip:witch", Label: "今晚不用药"})
	return actions
}

func hunterShotActions(room *Room, actor *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{{ID: "shoot:skip", Label: "不开枪"}}
	for _, target := range room.Players {
		if !target.Alive || target.ID == actor.ID {
			continue
		}
		actions = append(actions, aiplayer.LegalAction{
			ID:          "shoot:" + target.ID,
			Label:       fmt.Sprintf("开枪带走 %s", target.Name),
			Description: fmt.Sprintf("座位 %d 的存活玩家", target.Seat+1),
		})
	}
	return actions
}

func currentWerewolfKillTarget(room *Room) string {
	return mostVotedTarget(room.Werewolf.NightActions, func(playerID string) bool {
		player := findPlayerByID(room, playerID)
		return player != nil && player.Role == RoleWerewolf
	})
}

func werewolfVoteActions(room *Room, actor *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	for _, target := range room.Players {
		if !target.Alive || target.ID == actor.ID {
			continue
		}
		actions = append(actions, aiplayer.LegalAction{
			ID:          "vote:" + target.ID,
			Label:       fmt.Sprintf("投票给 %s", target.Name),
			Description: fmt.Sprintf("座位 %d 的存活玩家", target.Seat+1),
		})
	}
	return actions
}

func werewolfAIState(room *Room, actor *Player) map[string]any {
	players := make([]map[string]any, 0, len(room.Players))
	visibleAllies := []string{}
	for _, player := range room.Players {
		entry := map[string]any{
			"id":    player.ID,
			"name":  player.Name,
			"seat":  player.Seat,
			"alive": player.Alive,
		}
		if roleVisible(room, actor, player) {
			entry["role"] = player.Role
			entry["alignment"] = player.Alignment
			if player.ID != actor.ID && player.Alignment == actor.Alignment {
				visibleAllies = append(visibleAllies, player.ID)
			}
		}
		players = append(players, entry)
	}
	state := map[string]any{
		"phase":           room.Phase,
		"day":             room.Werewolf.Day,
		"yourRole":        actor.Role,
		"yourAlignment":   actor.Alignment,
		"players":         players,
		"visibleAllies":   visibleAllies,
		"lastNight":       room.Werewolf.LastNight,
		"votes":           cloneStringMap(room.Werewolf.Votes),
		"recentSpeech":    room.Speeches,
		"revealedIdiots":  cloneBoolMap(room.Werewolf.RevealedIdiots),
		"hunterPendingId": room.Werewolf.HunterPendingID,
	}
	if actor.Role == RoleSeer {
		state["seerChecks"] = cloneAlignmentMap(room.Werewolf.SeerChecks)
	}
	if actor.Role == RoleWitch {
		state["witch"] = map[string]any{
			"victimId":     currentWerewolfKillTarget(room),
			"antidoteUsed": room.Werewolf.WitchAntidoteUsed,
			"poisonUsed":   room.Werewolf.WitchPoisonUsed,
		}
	}
	return state
}

func avalonAIState(room *Room, actor *Player, phase string) map[string]any {
	players := make([]map[string]any, 0, len(room.Players))
	for _, player := range room.Players {
		entry := map[string]any{
			"id":    player.ID,
			"name":  player.Name,
			"seat":  player.Seat,
			"alive": player.Alive,
		}
		if roleVisible(room, actor, player) {
			entry["role"] = player.Role
			entry["alignment"] = player.Alignment
		}
		players = append(players, entry)
	}
	return map[string]any{
		"phase":         phase,
		"round":         room.Avalon.Round,
		"yourRole":      actor.Role,
		"yourAlignment": actor.Alignment,
		"players":       players,
		"leaderId":      room.Avalon.LeaderID,
		"team":          append([]string{}, room.Avalon.Team...),
		"teamVotes":     cloneBoolMap(room.Avalon.TeamVotes),
		"questResults":  append([]AvalonQuestResult{}, room.Avalon.QuestResults...),
		"successes":     room.Avalon.Successes,
		"fails":         room.Avalon.Fails,
		"recentSpeech":  room.Speeches,
	}
}

func undercoverAIState(room *Room, player *Player, phase string) map[string]any {
	return map[string]any{
		"phase":        phase,
		"round":        room.Undercover.Round,
		"yourRole":     player.Role,
		"yourWord":     undercoverWordForPlayer(room, player),
		"players":      publicPlayersForAI(room, player),
		"described":    cloneBoolMap(room.Undercover.Described),
		"votes":        cloneStringMap(room.Undercover.Votes),
		"recentSpeech": room.Speeches,
	}
}

func publicPlayersForAI(room *Room, actor *Player) []map[string]any {
	players := make([]map[string]any, 0, len(room.Players))
	for _, player := range room.Players {
		entry := map[string]any{
			"id":    player.ID,
			"name":  player.Name,
			"seat":  player.Seat,
			"alive": player.Alive,
		}
		if player.ID == actor.ID {
			entry["role"] = player.Role
			entry["alignment"] = player.Alignment
		}
		players = append(players, entry)
	}
	return players
}

func playerFromAction(room *Room, actionID string, prefix string) *Player {
	if !strings.HasPrefix(actionID, prefix) {
		return nil
	}
	return findPlayerByID(room, strings.TrimPrefix(actionID, prefix))
}

func (m *Manager) chooseAvalonTeam(room *Room, leader *Player) ([]string, string) {
	actions := avalonTeamActions(room, leader)
	if m.canUseLLM(leader) && len(actions) > 0 {
		decision, err := m.socialDecision(room, leader, avalonAIState(room, leader, "team"), actions)
		if err == nil && strings.HasPrefix(decision.ActionID, "team:") {
			team := strings.Split(strings.TrimPrefix(decision.ActionID, "team:"), ",")
			if len(team) == room.Avalon.RequiredTeam {
				return team, strings.TrimSpace(decision.Speech)
			}
		}
		if err != nil {
			slog.Warn("avalon llm team failed", "room", room.ID, "player", leader.ID, "playerName", leader.Name, "error", err)
		}
	}
	return nil, ""
}

func (m *Manager) chooseAvalonTeamVote(room *Room, player *Player) (bool, string, bool) {
	actions := []aiplayer.LegalAction{
		{ID: "vote:approve", Label: "同意这支队伍"},
		{ID: "vote:reject", Label: "反对这支队伍"},
	}
	if m.canUseLLM(player) {
		decision, err := m.socialDecision(room, player, avalonAIState(room, player, "team_vote"), actions)
		if err == nil {
			return decision.ActionID == "vote:approve", strings.TrimSpace(decision.Speech), true
		}
		slog.Warn("avalon llm team vote failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
	}
	return false, "", false
}

func (m *Manager) chooseAvalonQuestCard(room *Room, player *Player) (string, string) {
	actions := []aiplayer.LegalAction{{ID: "quest:success", Label: "提交成功牌"}}
	if player.Alignment == AlignmentEvil {
		actions = append(actions, aiplayer.LegalAction{ID: "quest:fail", Label: "提交失败牌"})
	}
	if m.canUseLLM(player) {
		decision, err := m.socialDecision(room, player, avalonAIState(room, player, "quest"), actions)
		if err == nil && decision.ActionID == "quest:fail" && player.Alignment == AlignmentEvil {
			return "fail", strings.TrimSpace(decision.Speech)
		}
		if err == nil {
			return "success", strings.TrimSpace(decision.Speech)
		}
		slog.Warn("avalon llm quest failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
	}
	return "", ""
}

func (m *Manager) chooseAvalonAssassination(room *Room, player *Player) (*Player, string) {
	actions := []aiplayer.LegalAction{}
	for _, target := range goodPlayers(room) {
		actions = append(actions, aiplayer.LegalAction{ID: "assassinate:" + target.ID, Label: fmt.Sprintf("刺杀 %s", target.Name)})
	}
	if m.canUseLLM(player) && len(actions) > 0 {
		decision, err := m.socialDecision(room, player, avalonAIState(room, player, "assassination"), actions)
		if err == nil {
			if target := playerFromAction(room, decision.ActionID, "assassinate:"); target != nil {
				return target, strings.TrimSpace(decision.Speech)
			}
		}
		if err != nil {
			slog.Warn("avalon llm assassination failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
		}
	}
	return nil, ""
}

func avalonTeamActions(room *Room, leader *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	players := append([]*Player{}, room.Players...)
	for _, first := range players {
		for _, second := range players {
			if room.Avalon.RequiredTeam == 2 {
				team := uniqueTeam([]string{first.ID, second.ID})
				if len(team) == 2 && slices.Contains(team, leader.ID) {
					actions = append(actions, avalonTeamAction(room, team))
				}
				continue
			}
			for _, third := range players {
				team := uniqueTeam([]string{leader.ID, first.ID, second.ID, third.ID})
				if len(team) == room.Avalon.RequiredTeam {
					actions = append(actions, avalonTeamAction(room, team))
				}
				if len(actions) >= 24 {
					return actions
				}
			}
		}
	}
	if len(actions) == 0 {
		actions = append(actions, avalonTeamAction(room, []string{leader.ID}))
	}
	return actions
}

func avalonTeamAction(room *Room, team []string) aiplayer.LegalAction {
	names := []string{}
	for _, id := range team {
		if player := findPlayerByID(room, id); player != nil {
			names = append(names, player.Name)
		}
	}
	return aiplayer.LegalAction{ID: "team:" + strings.Join(team, ","), Label: "提名 " + strings.Join(names, "、")}
}

func uniqueTeam(ids []string) []string {
	seen := map[string]bool{}
	team := []string{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		team = append(team, id)
	}
	return team
}

func (m *Manager) chooseUndercoverDescription(room *Room, player *Player) string {
	actions := undercoverDescriptionActions(room, player)
	if len(actions) == 0 {
		return ""
	}
	if player.AI != nil && player.AI.Level == string(aiplayer.LevelLLM) && m.aiProvider != nil && m.aiProvider.Enabled() {
		decision, err := m.socialDecision(room, player, undercoverAIState(room, player, "describe"), actions)
		if err == nil {
			if text := actionLabel(decision.ActionID, actions); text != "" {
				return text
			}
		} else {
			slog.Warn("undercover llm describe failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
		}
	}
	return ""
}

func (m *Manager) chooseUndercoverVote(room *Room, player *Player) *Player {
	actions := undercoverVoteActions(room, player)
	if len(actions) == 0 {
		return nil
	}
	if player.AI != nil && player.AI.Level == string(aiplayer.LevelLLM) && m.aiProvider != nil && m.aiProvider.Enabled() {
		decision, err := m.socialDecision(room, player, undercoverAIState(room, player, "vote"), actions)
		if err == nil && strings.HasPrefix(decision.ActionID, "vote:") {
			return findPlayerByID(room, strings.TrimPrefix(decision.ActionID, "vote:"))
		}
		if err != nil {
			slog.Warn("undercover llm vote failed", "room", room.ID, "player", player.ID, "playerName", player.Name, "error", err)
		}
	}
	return nil
}

func goodPlayers(room *Room) []*Player {
	players := []*Player{}
	for _, player := range room.Players {
		if player.Alignment == AlignmentGood {
			players = append(players, player)
		}
	}
	return players
}

func livingCount(room *Room) int {
	count := 0
	for _, player := range room.Players {
		if player.Alive {
			count++
		}
	}
	return count
}

func mostVotedTarget(votes map[string]string, actorFilter func(string) bool) string {
	counts := map[string]int{}
	bestID := ""
	bestCount := 0
	for actorID, targetID := range votes {
		if !actorFilter(actorID) {
			continue
		}
		counts[targetID]++
		if counts[targetID] > bestCount {
			bestID = targetID
			bestCount = counts[targetID]
		}
	}
	return bestID
}

func shuffledPlayers(players []*Player) []*Player {
	next := append([]*Player{}, players...)
	rand.Shuffle(len(next), func(i, j int) { next[i], next[j] = next[j], next[i] })
	return next
}

func createHumanPlayer(user UserView, role string, seat int) *Player {
	return &Player{
		ID:        "plr_" + randomToken(8),
		UserID:    user.ID,
		Name:      user.DisplayName,
		Seat:      seat,
		RoomRole:  role,
		Kind:      user.Kind,
		Connected: true,
		Alive:     true,
		JoinedAt:  time.Now().UTC(),
	}
}

func findPlayerByUserID(room *Room, userID string) *Player {
	for _, player := range room.Players {
		if player.UserID == userID {
			return player
		}
	}
	return nil
}

func findPlayerByID(room *Room, playerID string) *Player {
	for _, player := range room.Players {
		if player.ID == playerID {
			return player
		}
	}
	return nil
}

func nextAIProfile(room *Room, level aiplayer.Level) AIProfile {
	profile := aiplayer.NextProfile(usedAINames(room))
	return AIProfile{Name: profile.Name, Personality: profile.Personality, SpeechStyle: profile.SpeechStyle, Level: string(level)}
}

func usedAINames(room *Room) map[string]bool {
	used := map[string]bool{}
	for _, player := range room.Players {
		used[player.Name] = true
	}
	return used
}

func recordAction(room *Room, action PublicAction) {
	room.ActionSeq++
	action.Seq = room.ActionSeq
	room.RecentActions = append(room.RecentActions, action)
	if len(room.RecentActions) > 8 {
		room.RecentActions = room.RecentActions[len(room.RecentActions)-8:]
	}
}

func recordSpeech(room *Room, player *Player, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	runes := []rune(text)
	if len(runes) > 120 {
		text = string(runes[:120])
	}
	room.Speeches = append(room.Speeches, SpeechEntry{
		ID:         "speech_" + randomToken(8),
		PlayerID:   player.ID,
		PlayerName: player.Name,
		Text:       text,
		SpokenAt:   time.Now().UTC(),
	})
	if len(room.Speeches) > 18 {
		room.Speeches = room.Speeches[len(room.Speeches)-18:]
	}
	return true
}

func playerNote(room *Room, viewer *Player, targetID string) string {
	if viewer == nil || room.PlayerNotes == nil {
		return ""
	}
	return room.PlayerNotes[viewer.ID][targetID]
}

func setPlayerNote(room *Room, viewerID string, targetID string, note string) {
	note = roommeta.NormalizeNote(note)
	if room.PlayerNotes == nil {
		room.PlayerNotes = map[string]map[string]string{}
	}
	if room.PlayerNotes[viewerID] == nil {
		room.PlayerNotes[viewerID] = map[string]string{}
	}
	if note == "" {
		delete(room.PlayerNotes[viewerID], targetID)
		return
	}
	room.PlayerNotes[viewerID][targetID] = note
}

func reconcileLobbyConfig(room *Room) {
	if room.Game == GameWerewolf {
		reconcileWerewolfConfig(room)
	}
	if room.Game == GameUndercover {
		applyDefaultUndercoverConfig(room)
	}
}

func applyDefaultUndercoverConfig(room *Room) {
	if room.Undercover.PresetID == "" {
		room.Undercover.PresetID = defaultUndercoverPresetID()
	}
	room.Undercover.Presets = undercoverPresets()
	room.Undercover.Described = map[string]bool{}
	room.Undercover.Votes = map[string]string{}
}

func defaultUndercoverPresetID() string {
	return "daily"
}

func undercoverPresets() []UndercoverPreset {
	return []UndercoverPreset{
		{ID: "daily", Name: "日常生活", Description: "生活里常见但容易混淆的词。", Pairs: []UndercoverWordPair{
			{ID: "daily-1", CivilianWord: "咖啡", UndercoverWord: "奶茶", Category: "饮品"},
			{ID: "daily-2", CivilianWord: "公交车", UndercoverWord: "地铁", Category: "交通"},
			{ID: "daily-3", CivilianWord: "雨伞", UndercoverWord: "遮阳伞", Category: "物品"},
			{ID: "daily-4", CivilianWord: "键盘", UndercoverWord: "钢琴", Category: "物品"},
			{ID: "daily-5", CivilianWord: "火锅", UndercoverWord: "麻辣烫", Category: "食物"},
			{ID: "daily-6", CivilianWord: "电影院", UndercoverWord: "剧院", Category: "地点"},
		}},
		{ID: "internet", Name: "网络热词", Description: "更适合熟人局的互联网语境题库。", Pairs: []UndercoverWordPair{
			{ID: "internet-1", CivilianWord: "弹幕", UndercoverWord: "评论区", Category: "网络"},
			{ID: "internet-2", CivilianWord: "直播", UndercoverWord: "短视频", Category: "网络"},
			{ID: "internet-3", CivilianWord: "表情包", UndercoverWord: "贴纸", Category: "网络"},
			{ID: "internet-4", CivilianWord: "摸鱼", UndercoverWord: "摆烂", Category: "网络"},
			{ID: "internet-5", CivilianWord: "热搜", UndercoverWord: "推荐页", Category: "网络"},
		}},
		{ID: "anime", Name: "轻二次元", Description: "偏 ACG 的非 IP 词库，不依赖具体版权角色。", Pairs: []UndercoverWordPair{
			{ID: "anime-1", CivilianWord: "魔法少女", UndercoverWord: "变身英雄", Category: "幻想"},
			{ID: "anime-2", CivilianWord: "社团活动", UndercoverWord: "校园祭", Category: "校园"},
			{ID: "anime-3", CivilianWord: "机甲", UndercoverWord: "机器人", Category: "科幻"},
			{ID: "anime-4", CivilianWord: "异世界", UndercoverWord: "平行宇宙", Category: "幻想"},
			{ID: "anime-5", CivilianWord: "必杀技", UndercoverWord: "连招", Category: "战斗"},
		}},
		{ID: "ai-curated", Name: "AI 推荐", Description: "按 AI 参与感设计的更抽象题库。", Pairs: []UndercoverWordPair{
			{ID: "ai-1", CivilianWord: "灵感", UndercoverWord: "直觉", Category: "抽象"},
			{ID: "ai-2", CivilianWord: "记忆", UndercoverWord: "回忆", Category: "抽象"},
			{ID: "ai-3", CivilianWord: "计划", UndercoverWord: "策略", Category: "抽象"},
			{ID: "ai-4", CivilianWord: "规则", UndercoverWord: "约定", Category: "抽象"},
			{ID: "ai-5", CivilianWord: "推理", UndercoverWord: "猜测", Category: "抽象"},
		}},
	}
}

func undercoverPresetExists(id string) bool {
	for _, preset := range undercoverPresets() {
		if preset.ID == id {
			return true
		}
	}
	return false
}

func undercoverPresetName(id string) string {
	for _, preset := range undercoverPresets() {
		if preset.ID == id {
			return preset.Name
		}
	}
	return undercoverPresets()[0].Name
}

func chooseUndercoverPair(presetID string) UndercoverWordPair {
	for _, preset := range undercoverPresets() {
		if preset.ID == presetID && len(preset.Pairs) > 0 {
			return preset.Pairs[rand.IntN(len(preset.Pairs))]
		}
	}
	preset := undercoverPresets()[0]
	return preset.Pairs[rand.IntN(len(preset.Pairs))]
}

func undercoverCountForPlayers(count int) int {
	if count >= 7 {
		return 2
	}
	return 1
}

func firstLivingPlayerID(room *Room) string {
	for _, player := range room.Players {
		if player.Alive {
			return player.ID
		}
	}
	return ""
}

func nextUndescribedLivingPlayer(room *Room) *Player {
	for _, player := range room.Players {
		if player.Alive && !room.Undercover.Described[player.ID] {
			return player
		}
	}
	return nil
}

func mostVotedUndercoverTarget(votes map[string]string) (string, bool) {
	counts := map[string]int{}
	bestID := ""
	bestCount := 0
	tied := false
	for _, targetID := range votes {
		counts[targetID]++
		switch {
		case counts[targetID] > bestCount:
			bestID = targetID
			bestCount = counts[targetID]
			tied = false
		case counts[targetID] == bestCount:
			tied = true
		}
	}
	return bestID, tied
}

func undercoverWordForPlayer(room *Room, player *Player) string {
	switch player.Role {
	case RoleUndercover:
		return room.Undercover.WordPair.UndercoverWord
	case RoleBlank:
		return ""
	default:
		return room.Undercover.WordPair.CivilianWord
	}
}

func undercoverViewForViewer(room *Room, viewer *Player) UndercoverView {
	view := UndercoverView{
		Round:            room.Undercover.Round,
		PresetID:         room.Undercover.PresetID,
		IncludeBlank:     room.Undercover.IncludeBlank,
		CurrentSpeakerID: room.Undercover.CurrentSpeakerID,
		Described:        cloneBoolMap(room.Undercover.Described),
		Votes:            cloneStringMap(room.Undercover.Votes),
		LastEliminatedID: room.Undercover.LastEliminatedID,
	}
	if room.Phase == PhaseLobby {
		view.Presets = undercoverPresets()
		return view
	}
	if room.Phase == PhaseFinished {
		view.WordPair = room.Undercover.WordPair
		return view
	}
	if viewer != nil {
		view.WordPair = UndercoverWordPair{ID: room.Undercover.WordPair.ID, Category: room.Undercover.WordPair.Category}
		if viewer.Role == RoleUndercover {
			view.WordPair.UndercoverWord = room.Undercover.WordPair.UndercoverWord
		} else if viewer.Role == RoleCivilian {
			view.WordPair.CivilianWord = room.Undercover.WordPair.CivilianWord
		}
	}
	return view
}

func undercoverDescriptionActions(room *Room, player *Player) []aiplayer.LegalAction {
	word := undercoverWordForPlayer(room, player)
	if word == "" {
		return []aiplayer.LegalAction{
			{ID: "say:blank-soft", Label: "这个词和日常体验有关。"},
			{ID: "say:blank-object", Label: "它应该是大家都见过的东西。"},
			{ID: "say:blank-scene", Label: "我会先从使用场景判断。"},
		}
	}
	return []aiplayer.LegalAction{
		{ID: "say:scene", Label: fmt.Sprintf("%s一般会出现在具体场景里。", word)},
		{ID: "say:common", Label: fmt.Sprintf("我觉得%s挺常见。", word)},
		{ID: "say:feature", Label: fmt.Sprintf("%s的特点不能说太细。", word)},
	}
}

func actionLabel(actionID string, actions []aiplayer.LegalAction) string {
	for _, action := range actions {
		if action.ID == actionID {
			return action.Label
		}
	}
	return ""
}

func undercoverVoteActions(room *Room, player *Player) []aiplayer.LegalAction {
	actions := []aiplayer.LegalAction{}
	for _, target := range room.Players {
		if target.Alive && target.ID != player.ID {
			actions = append(actions, aiplayer.LegalAction{
				ID:          "vote:" + target.ID,
				Label:       target.Name,
				Description: fmt.Sprintf("投票给 %s", target.Name),
			})
		}
	}
	return actions
}

func createLog(text string) LogEntry {
	return LogEntry{ID: "log_" + randomToken(8), Text: text}
}

func createRoomID(game GameKind) string {
	prefix := "AVL"
	if game == GameWerewolf {
		prefix = "WWF"
	}
	if game == GameUndercover {
		prefix = "UND"
	}
	return prefix + randomToken(5)
}

func randomToken(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var builder strings.Builder
	for range length {
		builder.WriteByte(alphabet[rand.IntN(len(alphabet))])
	}
	return builder.String()
}

func hasDuplicate(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}

func cloneStringMap(source map[string]string) map[string]string {
	next := map[string]string{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	next := map[string]bool{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func cloneAlignmentMap(source map[string]Alignment) map[string]Alignment {
	next := map[string]Alignment{}
	for key, value := range source {
		next[key] = value
	}
	return next
}

func seerChecksForViewer(room *Room, viewer *Player) map[string]Alignment {
	if viewer == nil || viewer.Role != RoleSeer {
		return nil
	}
	next := map[string]Alignment{}
	for targetID, alignment := range room.Werewolf.SeerChecks {
		next[targetID] = alignment
	}
	return next
}
