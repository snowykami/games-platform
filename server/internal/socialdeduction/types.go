package socialdeduction

import "time"

type GameKind string

const (
	GameWerewolf   GameKind = "werewolf"
	GameAvalon     GameKind = "avalon"
	GameUndercover GameKind = "undercover"
)

type Phase string

const (
	PhaseLobby              Phase = "lobby"
	PhaseWerewolfNight      Phase = "night"
	PhaseWerewolfDay        Phase = "day"
	PhaseWerewolfVote       Phase = "vote"
	PhaseWerewolfHunter     Phase = "hunter"
	PhaseAvalonTeam         Phase = "team"
	PhaseAvalonVote         Phase = "team_vote"
	PhaseAvalonQuest        Phase = "quest"
	PhaseAssassination      Phase = "assassination"
	PhaseUndercoverDescribe Phase = "describe"
	PhaseUndercoverVote     Phase = "undercover_vote"
	PhaseFinished           Phase = "finished"
)

type Alignment string

const (
	AlignmentGood    Alignment = "good"
	AlignmentEvil    Alignment = "evil"
	AlignmentNeutral Alignment = "neutral"
)

type Role string

const (
	RoleVillager   Role = "villager"
	RoleWerewolf   Role = "werewolf"
	RoleSeer       Role = "seer"
	RoleGuard      Role = "guard"
	RoleWitch      Role = "witch"
	RoleHunter     Role = "hunter"
	RoleIdiot      Role = "idiot"
	RoleMerlin     Role = "merlin"
	RoleAssassin   Role = "assassin"
	RoleMinion     Role = "minion"
	RoleLoyal      Role = "loyal"
	RoleCivilian   Role = "civilian"
	RoleUndercover Role = "undercover"
	RoleBlank      Role = "blank"
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
	Seat           int        `json:"seat"`
	RoomRole       string     `json:"roomRole"`
	Kind           string     `json:"kind"`
	IsAI           bool       `json:"isAI"`
	Connected      bool       `json:"connected"`
	DisconnectedAt *time.Time `json:"disconnectedAt,omitempty"`
	AI             *AIProfile `json:"ai,omitempty"`
	Alive          bool       `json:"alive"`
	Role           Role       `json:"role,omitempty"`
	Alignment      Alignment  `json:"alignment,omitempty"`
	JoinedAt       time.Time  `json:"joinedAt"`
}

type PublicPlayer struct {
	ID             string     `json:"id"`
	UserID         string     `json:"-"`
	Name           string     `json:"name"`
	Seat           int        `json:"seat"`
	RoomRole       string     `json:"roomRole"`
	Kind           string     `json:"kind"`
	IsAI           bool       `json:"isAI"`
	Connected      bool       `json:"connected"`
	DisconnectedAt *time.Time `json:"disconnectedAt,omitempty"`
	AI             *AIProfile `json:"ai,omitempty"`
	Alive          bool       `json:"alive"`
	Role           Role       `json:"role,omitempty"`
	Alignment      Alignment  `json:"alignment,omitempty"`
	VisibleToYou   bool       `json:"visibleToYou"`
	Note           string     `json:"note,omitempty"`
}

type LogEntry struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	PlayerID   string `json:"playerId,omitempty"`
	PlayerName string `json:"playerName,omitempty"`
}

type SpeechEntry struct {
	ID         string    `json:"id"`
	PlayerID   string    `json:"playerId"`
	PlayerName string    `json:"playerName"`
	Text       string    `json:"text"`
	SpokenAt   time.Time `json:"spokenAt"`
}

type PublicAction struct {
	Seq       int    `json:"seq"`
	Type      string `json:"type"`
	ActorID   string `json:"actorId,omitempty"`
	ActorName string `json:"actorName,omitempty"`
	TargetID  string `json:"targetId,omitempty"`
	Message   string `json:"message"`
}

type WerewolfState struct {
	Day               int                           `json:"day"`
	RoleConfig        WerewolfRoleConfig            `json:"roleConfig"`
	RolePresets       []WerewolfRolePreset          `json:"rolePresets,omitempty"`
	NightActions      map[string]string             `json:"-"`
	SeerChecks        map[string]Alignment          `json:"-"`
	Votes             map[string]WerewolfVoteIntent `json:"votes"`
	DaySpeakers       map[string]bool               `json:"-"`
	LastNight         string                        `json:"lastNight,omitempty"`
	WitchAntidoteUsed bool                          `json:"-"`
	WitchPoisonUsed   bool                          `json:"-"`
	WitchSaveTargetID string                        `json:"-"`
	WitchPoisonID     string                        `json:"-"`
	RevealedIdiots    map[string]bool               `json:"-"`
	HunterPendingID   string                        `json:"-"`
	HunterAfterPhase  Phase                         `json:"-"`
}

type WerewolfVoteIntent struct {
	TargetID  string `json:"targetId"`
	Confirmed bool   `json:"confirmed"`
}

type WerewolfRoleCounts struct {
	Villager int `json:"villager"`
	Werewolf int `json:"werewolf"`
	Seer     int `json:"seer"`
	Guard    int `json:"guard"`
	Witch    int `json:"witch"`
	Hunter   int `json:"hunter"`
	Idiot    int `json:"idiot"`
}

type WerewolfRoleConfig struct {
	Mode        string             `json:"mode"`
	PresetID    string             `json:"presetId,omitempty"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Counts      WerewolfRoleCounts `json:"counts"`
}

type WerewolfRolePreset struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Players     int                `json:"players"`
	Counts      WerewolfRoleCounts `json:"counts"`
}

type AvalonState struct {
	Round         int                 `json:"round"`
	LeaderID      string              `json:"leaderId,omitempty"`
	Team          []string            `json:"team"`
	TeamVotes     map[string]bool     `json:"teamVotes"`
	QuestCards    map[string]string   `json:"-"`
	QuestResults  []AvalonQuestResult `json:"questResults"`
	RejectedTeams int                 `json:"rejectedTeams"`
	RequiredTeam  int                 `json:"requiredTeam"`
	RequiredFails int                 `json:"requiredFails"`
	Successes     int                 `json:"successes"`
	Fails         int                 `json:"fails"`
}

type UndercoverState struct {
	Round            int                             `json:"round"`
	PresetID         string                          `json:"presetId"`
	Presets          []UndercoverPreset              `json:"presets,omitempty"`
	WordPair         UndercoverWordPair              `json:"wordPair,omitempty"`
	IncludeBlank     bool                            `json:"includeBlank"`
	CurrentSpeakerID string                          `json:"currentSpeakerId,omitempty"`
	Described        map[string]bool                 `json:"described"`
	Votes            map[string]UndercoverVoteIntent `json:"votes"`
	LastEliminatedID string                          `json:"lastEliminatedId,omitempty"`
}

type UndercoverVoteIntent struct {
	TargetID  string `json:"targetId"`
	Confirmed bool   `json:"confirmed"`
}

type UndercoverPreset struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Pairs       []UndercoverWordPair `json:"pairs,omitempty"`
}

type UndercoverWordPair struct {
	ID             string `json:"id"`
	CivilianWord   string `json:"civilianWord,omitempty"`
	UndercoverWord string `json:"undercoverWord,omitempty"`
	Category       string `json:"category,omitempty"`
}

type AvalonQuestResult struct {
	Round     int `json:"round"`
	TeamSize  int `json:"teamSize"`
	FailCards int `json:"failCards"`
}

type Room struct {
	ID                   string
	Game                 GameKind
	HostUserID           string
	Phase                Phase
	Players              []*Player
	Werewolf             WerewolfState
	Avalon               AvalonState
	Undercover           UndercoverState
	Winner               Alignment
	WinnerMessage        string
	Log                  []LogEntry
	Speeches             []SpeechEntry
	LastAISpeechSourceID string
	PlayerNotes          map[string]map[string]string
	ActionSeq            int
	RecentActions        []PublicAction
	CreatedAt            time.Time
	UpdatedAt            time.Time
	RuleUpdatedAt        time.Time
	SpeechUpdatedAt      time.Time
	PresenceUpdatedAt    time.Time
}

type PublicRoom struct {
	ID               string         `json:"id"`
	Game             GameKind       `json:"game"`
	HostUserID       string         `json:"-"`
	HostPlayerID     string         `json:"hostPlayerId,omitempty"`
	Phase            Phase          `json:"phase"`
	Players          []PublicPlayer `json:"players"`
	YouPlayerID      string         `json:"youPlayerId,omitempty"`
	MinPlayers       int            `json:"minPlayers"`
	MaxPlayers       int            `json:"maxPlayers"`
	GodViewAvailable bool           `json:"godViewAvailable,omitempty"`
	GodViewEnabled   bool           `json:"godViewEnabled,omitempty"`
	Werewolf         WerewolfView   `json:"werewolf,omitempty"`
	Avalon           AvalonView     `json:"avalon,omitempty"`
	Undercover       UndercoverView `json:"undercover,omitempty"`
	Winner           Alignment      `json:"winner,omitempty"`
	WinnerMessage    string         `json:"winnerMessage,omitempty"`
	Log              []LogEntry     `json:"log"`
	Speeches         []SpeechEntry  `json:"speeches"`
	ActionSeq        int            `json:"actionSeq"`
	RecentActions    []PublicAction `json:"recentActions"`
}

type WerewolfView struct {
	Day               int                           `json:"day"`
	RoleConfig        WerewolfRoleConfig            `json:"roleConfig"`
	RolePresets       []WerewolfRolePreset          `json:"rolePresets,omitempty"`
	SeerChecks        map[string]Alignment          `json:"seerChecks,omitempty"`
	Votes             map[string]WerewolfVoteIntent `json:"votes"`
	DaySpeakers       map[string]bool               `json:"daySpeakers,omitempty"`
	NightSubmitted    bool                          `json:"nightActionSubmitted,omitempty"`
	LastNight         string                        `json:"lastNight,omitempty"`
	WitchVictimID     string                        `json:"witchVictimId,omitempty"`
	WitchAntidoteUsed bool                          `json:"witchAntidoteUsed,omitempty"`
	WitchPoisonUsed   bool                          `json:"witchPoisonUsed,omitempty"`
	HunterPendingID   string                        `json:"hunterPendingId,omitempty"`
	RevealedIdiots    map[string]bool               `json:"revealedIdiots,omitempty"`
}

type AvalonView struct {
	Round         int                 `json:"round"`
	LeaderID      string              `json:"leaderId,omitempty"`
	Team          []string            `json:"team"`
	TeamVotes     map[string]bool     `json:"teamVotes"`
	TeamVoteCount int                 `json:"teamVoteCount"`
	QuestResults  []AvalonQuestResult `json:"questResults"`
	RejectedTeams int                 `json:"rejectedTeams"`
	RequiredTeam  int                 `json:"requiredTeam"`
	RequiredFails int                 `json:"requiredFails"`
	Successes     int                 `json:"successes"`
	Fails         int                 `json:"fails"`
}

type UndercoverView struct {
	Round            int                             `json:"round"`
	PresetID         string                          `json:"presetId"`
	Presets          []UndercoverPreset              `json:"presets,omitempty"`
	WordPair         UndercoverWordPair              `json:"wordPair,omitempty"`
	IncludeBlank     bool                            `json:"includeBlank"`
	CurrentSpeakerID string                          `json:"currentSpeakerId,omitempty"`
	Described        map[string]bool                 `json:"described"`
	Votes            map[string]UndercoverVoteIntent `json:"votes"`
	LastEliminatedID string                          `json:"lastEliminatedId,omitempty"`
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
