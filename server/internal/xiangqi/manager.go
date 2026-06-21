package xiangqi

import (
	"sync"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

const (
	minPlayers = 2
	maxPlayers = 2
)

var slidingDirections = []Position{
	{X: 1, Y: 0},
	{X: -1, Y: 0},
	{X: 0, Y: 1},
	{X: 0, Y: -1},
}

type Manager struct {
	*gameactor.RoomRuntime

	mu         sync.Mutex
	rooms      map[string]*Room
	aiProvider aiplayer.Provider

	aiController *aiagent.Controller
}

func NewManager(aiProvider aiplayer.Provider) *Manager {
	return &Manager{
		RoomRuntime:  gameactor.NewRoomRuntime(64),
		rooms:        map[string]*Room{},
		aiProvider:   aiProvider,
		aiController: aiagent.NewController("xiangqi", aiProvider, aiplayer.DecisionTimeout),
	}
}
