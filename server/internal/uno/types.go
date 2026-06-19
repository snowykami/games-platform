package uno

import "time"

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
}

type Player struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Kind      string     `json:"kind"`
	IsAI      bool       `json:"isAI"`
	Connected bool       `json:"connected"`
	AI        *AIProfile `json:"ai,omitempty"`
	Hand      []Card     `json:"hand,omitempty"`
	HandCount int        `json:"handCount"`
	JoinedAt  time.Time  `json:"joinedAt"`
}

type LogEntry struct {
	ID   string `json:"id"`
	Text string `json:"text"`
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
	ID                 string
	HostUserID         string
	VariantKey         string
	ThemeKey           string
	Phase              Phase
	Players            []*Player
	DrawPile           []Card
	DiscardPile        []Card
	CurrentPlayerIndex int
	Direction          int
	ActiveColor        Color
	WinnerID           string
	Log                []LogEntry
	ActionSeq          int
	RecentActions      []PublicAction
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PublicRoom struct {
	ID              string         `json:"id"`
	HostUserID      string         `json:"hostUserId"`
	VariantKey      string         `json:"variantKey"`
	ThemeKey        string         `json:"themeKey"`
	Phase           Phase          `json:"phase"`
	Players         []Player       `json:"players"`
	TopCard         *Card          `json:"topCard,omitempty"`
	DrawPileCount   int            `json:"drawPileCount"`
	CurrentPlayerID string         `json:"currentPlayerId,omitempty"`
	Direction       int            `json:"direction"`
	ActiveColor     Color          `json:"activeColor,omitempty"`
	PlayableCardIDs []string       `json:"playableCardIds"`
	WinnerID        string         `json:"winnerId,omitempty"`
	Log             []LogEntry     `json:"log"`
	ActionSeq       int            `json:"actionSeq"`
	RecentActions   []PublicAction `json:"recentActions"`
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
