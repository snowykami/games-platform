package gomoku

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

var directions = []Point{
	{X: 1, Y: 0},
	{X: 0, Y: 1},
	{X: 1, Y: 1},
	{X: 1, Y: -1},
}

type Manager struct {
	*gameactor.RoomRuntime

	aiProvider aiplayer.Provider
	mu         sync.Mutex
	rooms      map[string]*Room

	aiController *aiagent.Controller
}

func NewManager(aiProvider aiplayer.Provider) *Manager {
	return &Manager{
		RoomRuntime:  gameactor.NewRoomRuntime(64),
		aiProvider:   aiProvider,
		rooms:        map[string]*Room{},
		aiController: aiagent.NewController("gomoku", aiProvider, aiplayer.DecisionTimeout),
	}
}
