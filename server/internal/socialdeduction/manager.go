package socialdeduction

import (
	"sync"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

const (
	werewolfMinPlayers    = 6
	werewolfMaxPlayers    = 12
	avalonMinPlayers      = 5
	avalonMaxPlayers      = 10
	undercoverMinPlayers  = 4
	undercoverMaxPlayers  = 10
	socialDecisionTimeout = aiplayer.DecisionTimeout
)

type Manager struct {
	*gameactor.RoomRuntime

	aiProvider aiplayer.Provider
	game       GameKind
	mu         sync.Mutex
	rooms      map[string]*Room
	aiSessions map[string]*socialAISession

	aiController *aiagent.Controller
}

func NewManager(game GameKind, aiProvider aiplayer.Provider) *Manager {
	return &Manager{
		RoomRuntime:  gameactor.NewRoomRuntime(64),
		aiProvider:   aiProvider,
		aiSessions:   map[string]*socialAISession{},
		game:         game,
		rooms:        map[string]*Room{},
		aiController: aiagent.NewController(string(game), aiProvider, socialDecisionTimeout),
	}
}
