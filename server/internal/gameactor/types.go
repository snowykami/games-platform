package gameactor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

type EventLane string

const (
	LaneRule     EventLane = "rule"
	LaneSpeech   EventLane = "speech"
	LanePresence EventLane = "presence"
	LaneView     EventLane = "view"
)

type RoomVersion struct {
	RuleVersion     int64
	SpeechVersion   int64
	PresenceVersion int64
	ViewVersion     int64
}

func (v RoomVersion) Bump(lane EventLane) RoomVersion {
	switch lane {
	case LaneRule:
		v.RuleVersion++
		v.ViewVersion++
	case LaneSpeech:
		v.SpeechVersion++
		v.ViewVersion++
	case LanePresence:
		v.PresenceVersion++
		v.ViewVersion++
	case LaneView:
		v.ViewVersion++
	}
	return v
}

type RoomEventType string

const (
	EventHumanIntentSubmitted RoomEventType = "human_intent_submitted"
	EventAIIntentSubmitted    RoomEventType = "ai_intent_submitted"
	EventPlayerSpeech         RoomEventType = "player_speech_submitted"
	EventPlayerConnected      RoomEventType = "player_connected"
	EventPlayerDisconnected   RoomEventType = "player_disconnected"
	EventTurnDeadlineReached  RoomEventType = "turn_deadline_reached"
	EventRoomIdleTimeout      RoomEventType = "room_idle_timeout_reached"
	EventAgentRequestTimeout  RoomEventType = "agent_request_timed_out"
	EventRoomClosed           RoomEventType = "room_closed"
)

type RoomEvent struct {
	ID        string
	RoomID    string
	PlayerID  string
	Type      RoomEventType
	Lane      EventLane
	Payload   any
	CreatedAt time.Time
}

type AgentEventType string

const (
	AgentObserve            AgentEventType = "observe"
	AgentRequiredAction     AgentEventType = "required_action"
	AgentOptionalSpeech     AgentEventType = "optional_speech"
	AgentPhaseChanged       AgentEventType = "phase_changed"
	AgentPlayerSpeech       AgentEventType = "player_speech_observed"
	AgentPrivateInfoChanged AgentEventType = "private_info_changed"
	AgentShutdown           AgentEventType = "shutdown"
)

type AgentEvent struct {
	ID              string
	RoomID          string
	PlayerID        string
	Type            AgentEventType
	RoomVersion     RoomVersion
	Phase           string
	PublicState     any
	PrivateState    any
	RecentEvents    []RoomEvent
	LegalActions    []aiplayer.LegalAction
	LegalActionHash string
	Deadline        time.Time
}

type IntentKind string

const (
	IntentRequiredAction IntentKind = "required_action"
	IntentOptionalSpeech IntentKind = "optional_speech"
)

type PlayerIntent struct {
	RoomID    string
	PlayerID  string
	RequestID string
	Kind      IntentKind
	ActionID  string
	Speech    string
	Reason    string
	Notes     map[string]string
	CreatedAt time.Time
}

type ActiveAgentRequest struct {
	RequestID       string
	PlayerID        string
	Kind            AgentEventType
	Phase           string
	RuleVersion     int64
	SpeechVersion   int64
	LegalActionHash string
	Deadline        time.Time
}

func (r ActiveAgentRequest) StaleReason(current RoomVersion, phase string, playerID string, legalActions []aiplayer.LegalAction) string {
	if r.PlayerID != playerID {
		return "actor_changed"
	}
	if r.Phase != phase {
		return "phase_changed"
	}
	if !r.Deadline.IsZero() && time.Now().After(r.Deadline) {
		return "deadline_exceeded"
	}
	if r.Kind == AgentRequiredAction {
		if r.RuleVersion != current.RuleVersion {
			return "rule_version_changed"
		}
		if r.LegalActionHash != "" && r.LegalActionHash != LegalActionHash(legalActions) {
			return "legal_actions_changed"
		}
		return ""
	}
	if r.Kind == AgentOptionalSpeech && r.SpeechVersion != current.SpeechVersion {
		return "speech_version_changed"
	}
	return ""
}

func LegalActionHash(actions []aiplayer.LegalAction) string {
	payload, err := json.Marshal(actions)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
