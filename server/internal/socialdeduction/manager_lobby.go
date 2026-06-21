package socialdeduction

import (
	"errors"
	"fmt"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func (m *Manager) CreateRoom(user UserView) PublicRoom {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	room := &Room{
		ID:                createRoomID(m.game),
		Game:              m.game,
		HostUserID:        user.ID,
		Phase:             PhaseLobby,
		CreatedAt:         now,
		UpdatedAt:         now,
		RuleUpdatedAt:     now,
		SpeechUpdatedAt:   now,
		PresenceUpdatedAt: now,
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
	joined := false
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
		joined = true
	}

	player.Connected = true
	player.DisconnectedAt = nil
	if joined {
		touchRule(room)
	} else {
		touchPresence(room)
	}
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
		touchPresence(room)
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
	touchRule(room)
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
	touchView(room)
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
		if player.IsAI {
			m.removeSocialAgent(room.ID, player.ID)
		}
		for seat, nextPlayer := range room.Players {
			nextPlayer.Seat = seat
		}
		reconcileLobbyConfig(room)
		room.Log = append(room.Log, createLog(fmt.Sprintf("%s 被房主移出了房间。", player.Name)))
		touchRule(room)
		return m.publicRoom(room, actorID), nil
	}
	return PublicRoom{}, errors.New("player_not_found")
}

func (m *Manager) Close() {
	if m.aiController != nil {
		m.aiController.Close()
	}
	if m.RoomRuntime != nil {
		m.RoomRuntime.Close()
	}
	m.mu.Lock()
	roomIDs := make([]string, 0, len(m.rooms))
	for roomID := range m.rooms {
		roomIDs = append(roomIDs, roomID)
	}
	m.mu.Unlock()
	for _, roomID := range roomIDs {
		m.removeRoomAgents(roomID)
	}
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
	previousPhase := room.Phase
	if !recordSpeech(room, player, text) {
		return PublicRoom{}, errors.New("invalid_speech")
	}
	if room.Phase != previousPhase {
		touchRuleAndSpeech(room)
	} else {
		touchSpeech(room)
	}
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
	nextName, err := normalizePlayerDisplayName(displayName)
	if err != nil {
		return PublicRoom{}, err
	}
	oldName := player.Name
	player.Name = nextName
	room.Log = append(room.Log, createLog(fmt.Sprintf("%s 改名为 %s。", oldName, nextName)))
	touchView(room)
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
	touchView(room)
	return m.publicRoom(room, actorID), nil
}
