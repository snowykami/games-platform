package uno

import (
	"sync"
	"time"
)

type Color string

const (
	ColorRed    Color = "red"
	ColorYellow Color = "yellow"
	ColorGreen  Color = "green"
	ColorBlue   Color = "blue"
	ColorWild   Color = "wild"
)

type Kind string

const (
	KindNumber       Kind = "number"
	KindSkip         Kind = "skip"
	KindReverse      Kind = "reverse"
	KindDrawTwo      Kind = "draw-two"
	KindWild         Kind = "wild"
	KindWildDrawFour Kind = "wild-draw-four"
	KindWildDrawSix  Kind = "wild-draw-six"
	KindWildDrawTen  Kind = "wild-draw-ten"
	KindFlip         Kind = "flip"
)

type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhasePlaying  Phase = "playing"
	PhaseFinished Phase = "finished"
)

type Card struct {
	ID    string `json:"id"`
	Color Color  `json:"color"`
	Kind  Kind   `json:"kind"`
	Value *int   `json:"value,omitempty"`
}

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
	Hand           []Card     `json:"hand,omitempty"`
	HandCount      int        `json:"handCount"`
	NeedsUNO       bool       `json:"needsUno"`
	JoinedAt       time.Time  `json:"joinedAt"`
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
	ActionPlay   ActionType = "play"
	ActionDraw   ActionType = "draw"
	ActionEffect ActionType = "effect"
	ActionWin    ActionType = "win"
)

type PublicAction struct {
	Seq       int        `json:"seq"`
	Type      ActionType `json:"type"`
	ActorID   string     `json:"actorId"`
	ActorName string     `json:"actorName"`
	TargetID  string     `json:"targetId,omitempty"`
	Card      *Card      `json:"card,omitempty"`
	Count     int        `json:"count,omitempty"`
	Message   string     `json:"message"`
}

type Room struct {
	mu                    sync.Mutex
	ID                    string
	HostUserID            string
	VariantKey            string
	ThemeKey              string
	Phase                 Phase
	Players               []*Player
	DrawPile              []Card
	DiscardPile           []Card
	CurrentPlayerIndex    int
	Direction             int
	ActiveColor           Color
	PendingDrawCount      int
	PendingDrawKind       Kind
	FlipSide              bool
	Rules                 RuleSet
	WinnerID              string
	Log                   []LogEntry
	Speeches              []SpeechEntry
	LastAISpeechSourceID  string
	ActionSeq             int
	RecentActions         []PublicAction
	TurnDeadline          *time.Time
	AllHumansOfflineSince *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type PublicRoom struct {
	ID                   string         `json:"id"`
	HostUserID           string         `json:"-"`
	HostPlayerID         string         `json:"hostPlayerId,omitempty"`
	YouPlayerID          string         `json:"youPlayerId,omitempty"`
	VariantKey           string         `json:"variantKey"`
	ThemeKey             string         `json:"themeKey"`
	Phase                Phase          `json:"phase"`
	Players              []Player       `json:"players"`
	TopCard              *Card          `json:"topCard,omitempty"`
	DrawPileCount        int            `json:"drawPileCount"`
	CurrentPlayerID      string         `json:"currentPlayerId,omitempty"`
	Direction            int            `json:"direction"`
	ActiveColor          Color          `json:"activeColor,omitempty"`
	PendingDrawCount     int            `json:"pendingDrawCount"`
	FlipSide             bool           `json:"flipSide"`
	Rules                RuleSet        `json:"rules"`
	PlayableCardIDs      []string       `json:"playableCardIds"`
	WinnerID             string         `json:"winnerId,omitempty"`
	Log                  []LogEntry     `json:"log"`
	Speeches             []SpeechEntry  `json:"speeches"`
	ActionSeq            int            `json:"actionSeq"`
	RecentActions        []PublicAction `json:"recentActions"`
	TurnDeadline         *time.Time     `json:"turnDeadline,omitempty"`
	TurnRemainingSeconds int            `json:"turnRemainingSeconds"`
}

type UserView struct {
	ID          string
	DisplayName string
	Role        string
	Kind        string
}

type RoomOptions struct {
	VariantKey string `json:"variantKey"`
	ThemeKey   string `json:"themeKey"`
}

type AIOptions struct {
	Level string `json:"level"`
}

type RuleSet struct {
	Stacking  bool `json:"stacking"`
	SevenZero bool `json:"sevenZero"`
	JumpIn    bool `json:"jumpIn"`
	AllWild   bool `json:"allWild"`
	Flip      bool `json:"flip"`
	NoMercy   bool `json:"noMercy"`
}
