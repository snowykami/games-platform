package xiangqi

import "time"

const (
	BoardWidth  = 9
	BoardHeight = 10
)

type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhasePlaying  Phase = "playing"
	PhaseFinished Phase = "finished"
)

type Side string

const (
	SideRed   Side = "red"
	SideBlack Side = "black"
)

type PieceType string

const (
	PieceGeneral  PieceType = "general"
	PieceAdvisor  PieceType = "advisor"
	PieceElephant PieceType = "elephant"
	PieceHorse    PieceType = "horse"
	PieceRook     PieceType = "rook"
	PieceCannon   PieceType = "cannon"
	PieceSoldier  PieceType = "soldier"
)

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Piece struct {
	ID   string    `json:"id"`
	Side Side      `json:"side"`
	Type PieceType `json:"type"`
	X    int       `json:"x"`
	Y    int       `json:"y"`
}

type AIProfile struct {
	Name        string `json:"name"`
	Personality string `json:"personality"`
	SpeechStyle string `json:"speechStyle,omitempty"`
	Level       string `json:"level"`
}

type AIOptions struct {
	Level string `json:"level"`
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
	Side           Side       `json:"side,omitempty"`
	JoinedAt       time.Time  `json:"joinedAt"`
}

type Move struct {
	ID         string    `json:"id"`
	PieceID    string    `json:"pieceId"`
	PieceType  PieceType `json:"pieceType"`
	Side       Side      `json:"side"`
	From       Position  `json:"from"`
	To         Position  `json:"to"`
	Captured   *Piece    `json:"captured,omitempty"`
	Check      bool      `json:"check"`
	Checkmate  bool      `json:"checkmate"`
	PlayerID   string    `json:"playerId"`
	PlayerName string    `json:"playerName"`
	PlayedAt   time.Time `json:"playedAt"`
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
	ActionMove      ActionType = "move"
	ActionCapture   ActionType = "capture"
	ActionCheck     ActionType = "check"
	ActionCheckmate ActionType = "checkmate"
)

type PublicAction struct {
	Seq       int        `json:"seq"`
	Type      ActionType `json:"type"`
	ActorID   string     `json:"actorId"`
	ActorName string     `json:"actorName"`
	Move      *Move      `json:"move,omitempty"`
	Message   string     `json:"message"`
}

type Room struct {
	ID                   string
	HostUserID           string
	Phase                Phase
	Players              []*Player
	Pieces               []Piece
	Moves                []Move
	CurrentPlayerIndex   int
	WinnerID             string
	CheckSide            Side
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
	Pieces          []Piece        `json:"pieces"`
	Moves           []Move         `json:"moves"`
	CurrentPlayerID string         `json:"currentPlayerId,omitempty"`
	WinnerID        string         `json:"winnerId,omitempty"`
	CheckSide       Side           `json:"checkSide,omitempty"`
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
