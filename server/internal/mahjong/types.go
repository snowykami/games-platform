package mahjong

import "time"

type Phase string

const (
	PhaseLobby    Phase = "lobby"
	PhasePlaying  Phase = "playing"
	PhaseClaiming Phase = "claiming"
	PhaseFinished Phase = "finished"
)

type TileKind string

const (
	TileCharacters TileKind = "characters"
	TileDots       TileKind = "dots"
	TileBamboo     TileKind = "bamboo"
	TileWind       TileKind = "wind"
	TileDragon     TileKind = "dragon"
)

type Wind string

const (
	WindEast  Wind = "east"
	WindSouth Wind = "south"
	WindWest  Wind = "west"
	WindNorth Wind = "north"
)

type Dragon string

const (
	DragonRed   Dragon = "red"
	DragonGreen Dragon = "green"
	DragonWhite Dragon = "white"
)

type MeldKind string

const (
	MeldChow MeldKind = "chow"
	MeldPung MeldKind = "pung"
	MeldKong MeldKind = "kong"
)

type ClaimKind string

const (
	ClaimHu   ClaimKind = "hu"
	ClaimPeng ClaimKind = "peng"
	ClaimChi  ClaimKind = "chi"
)

type Tile struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	Kind   TileKind `json:"kind"`
	Rank   int      `json:"rank,omitempty"`
	Wind   Wind     `json:"wind,omitempty"`
	Dragon Dragon   `json:"dragon,omitempty"`
}

type Meld struct {
	ID           string   `json:"id"`
	Kind         MeldKind `json:"kind"`
	Tiles        []Tile   `json:"tiles"`
	FromPlayerID string   `json:"fromPlayerId,omitempty"`
	Exposed      bool     `json:"exposed"`
}

type FanPattern struct {
	Name string `json:"name"`
	Fan  int    `json:"fan"`
}

type WinResult struct {
	CanWin   bool         `json:"canWin"`
	Fan      int          `json:"fan"`
	Patterns []FanPattern `json:"patterns"`
	Reason   string       `json:"reason,omitempty"`
}

type AIProfile struct {
	Name        string `json:"name"`
	Personality string `json:"personality"`
	Level       string `json:"level"`
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
	Wind      Wind       `json:"wind"`
	Hand      []Tile     `json:"hand"`
	Melds     []Meld     `json:"melds"`
	Discards  []Tile     `json:"discards"`
	JoinedAt  time.Time  `json:"joinedAt"`
}

type PublicPlayer struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Kind      string     `json:"kind"`
	IsAI      bool       `json:"isAI"`
	Connected bool       `json:"connected"`
	AI        *AIProfile `json:"ai,omitempty"`
	Wind      Wind       `json:"wind"`
	Hand      []Tile     `json:"hand"`
	HandCount int        `json:"handCount"`
	Melds     []Meld     `json:"melds"`
	Discards  []Tile     `json:"discards"`
}

type ClaimOption struct {
	ID            string    `json:"id"`
	PlayerID      string    `json:"playerId"`
	Kind          ClaimKind `json:"kind"`
	Tile          Tile      `json:"tile"`
	TilesFromHand []Tile    `json:"tilesFromHand"`
	WinResult     WinResult `json:"winResult,omitempty"`
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
	ActionDraw    ActionType = "draw"
	ActionDiscard ActionType = "discard"
	ActionClaim   ActionType = "claim"
	ActionWin     ActionType = "win"
	ActionStart   ActionType = "start"
)

type PublicAction struct {
	Seq       int        `json:"seq"`
	Type      ActionType `json:"type"`
	ActorID   string     `json:"actorId"`
	ActorName string     `json:"actorName"`
	TargetID  string     `json:"targetId,omitempty"`
	Tile      *Tile      `json:"tile,omitempty"`
	Message   string     `json:"message"`
}

type RuleSet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MinFan      int    `json:"minFan"`
	Description string `json:"description"`
}

type LastDiscard struct {
	Tile     Tile   `json:"tile"`
	PlayerID string `json:"playerId"`
}

type Room struct {
	ID                 string
	HostUserID         string
	Phase              Phase
	Players            []*Player
	Wall               []Tile
	DeadWall           []Tile
	CurrentPlayerIndex int
	DealerIndex        int
	RoundWind          Wind
	HasDrawn           bool
	LastDiscard        *LastDiscard
	ClaimOptions       []ClaimOption
	RuleSet            RuleSet
	WinnerID           string
	WinResult          WinResult
	Log                []LogEntry
	Speeches           []SpeechEntry
	ActionSeq          int
	RecentActions      []PublicAction
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type PublicRoom struct {
	ID              string         `json:"id"`
	HostUserID      string         `json:"hostUserId"`
	Phase           Phase          `json:"phase"`
	Players         []PublicPlayer `json:"players"`
	WallCount       int            `json:"wallCount"`
	DeadWallCount   int            `json:"deadWallCount"`
	CurrentPlayerID string         `json:"currentPlayerId,omitempty"`
	DealerID        string         `json:"dealerId,omitempty"`
	RoundWind       Wind           `json:"roundWind"`
	HasDrawn        bool           `json:"hasDrawn"`
	LastDiscard     *LastDiscard   `json:"lastDiscard,omitempty"`
	ClaimOptions    []ClaimOption  `json:"claimOptions"`
	RuleSet         RuleSet        `json:"ruleset"`
	WinnerID        string         `json:"winnerId,omitempty"`
	WinResult       WinResult      `json:"winResult,omitempty"`
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
