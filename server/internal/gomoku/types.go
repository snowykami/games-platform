package gomoku

import "time"

const BoardSize = 15

type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhasePlaying  Phase = "playing"
	PhaseFinished Phase = "finished"
)

type Stone string

const (
	StoneBlack Stone = "black"
	StoneWhite Stone = "white"
)

type AIProfile struct {
	Name        string `json:"name"`
	Personality string `json:"personality"`
	SpeechStyle string `json:"speechStyle,omitempty"`
	Level       string `json:"level"`
}

type Player struct {
	ID             string     `json:"id"`
	UserID         string     `json:"-"`
	Name           string     `json:"name"`
	Role           string     `json:"role"`
	Kind           string     `json:"kind"`
	IsAI           bool       `json:"isAI"`
	Connected      bool       `json:"connected"`
	DisconnectedAt *time.Time `json:"disconnectedAt,omitempty"`
	AI             *AIProfile `json:"ai,omitempty"`
	Stone          Stone      `json:"stone,omitempty"`
	JoinedAt       time.Time  `json:"joinedAt"`
}

type Move struct {
	X          int       `json:"x"`
	Y          int       `json:"y"`
	Stone      Stone     `json:"stone"`
	PlayerID   string    `json:"playerId"`
	PlayerName string    `json:"playerName"`
	PlacedAt   time.Time `json:"placedAt"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type LogEntry struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

type SpeechEntry struct {
	ID         string    `json:"id"`
	PlayerID   string    `json:"playerId"`
	PlayerName string    `json:"playerName"`
	Text       string    `json:"text"`
	SpokenAt   time.Time `json:"spokenAt"`
}

type ActionType string

const (
	ActionPlace ActionType = "place"
	ActionWin   ActionType = "win"
	ActionDraw  ActionType = "draw"
)

type PublicAction struct {
	Seq       int        `json:"seq"`
	Type      ActionType `json:"type"`
	ActorID   string     `json:"actorId"`
	ActorName string     `json:"actorName"`
	X         int        `json:"x,omitempty"`
	Y         int        `json:"y,omitempty"`
	Stone     Stone      `json:"stone,omitempty"`
	Message   string     `json:"message"`
}

type Room struct {
	ID                   string
	HostUserID           string
	Phase                Phase
	Players              []*Player
	Board                [BoardSize][BoardSize]Stone
	Moves                []Move
	CurrentPlayerIndex   int
	WinnerID             string
	WinningLine          []Point
	IsDraw               bool
	Log                  []LogEntry
	Speeches             []SpeechEntry
	LastAISpeechSourceID string
	ActionSeq            int
	RecentActions        []PublicAction
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type PublicRoom struct {
	ID              string         `json:"id"`
	HostUserID      string         `json:"-"`
	HostPlayerID    string         `json:"hostPlayerId,omitempty"`
	YouPlayerID     string         `json:"youPlayerId,omitempty"`
	Phase           Phase          `json:"phase"`
	Players         []Player       `json:"players"`
	BoardSize       int            `json:"boardSize"`
	Moves           []Move         `json:"moves"`
	CurrentPlayerID string         `json:"currentPlayerId,omitempty"`
	WinnerID        string         `json:"winnerId,omitempty"`
	WinningLine     []Point        `json:"winningLine"`
	IsDraw          bool           `json:"isDraw"`
	Log             []LogEntry     `json:"log"`
	Speeches        []SpeechEntry  `json:"speeches"`
	ActionSeq       int            `json:"actionSeq"`
	RecentActions   []PublicAction `json:"recentActions"`
}

type UserView struct {
	ID          string
	DisplayName string
	Role        string
	Kind        string
}

type AIOptions struct {
	Level string `json:"level"`
}
