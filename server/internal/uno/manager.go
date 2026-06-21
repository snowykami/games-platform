package uno

import (
	"sync"
	"time"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

const (
	minPlayers     = 2
	maxPlayers     = 10
	turnTimeout    = 30 * time.Second
	offlineRoomTTL = 60 * time.Second
)

var colors = []Color{ColorRed, ColorYellow, ColorGreen, ColorBlue}

type Manager struct {
	*gameactor.RoomRuntime

	aiProvider aiplayer.Provider
	mu         sync.RWMutex
	rooms      map[string]*Room

	aiController *aiagent.Controller
}

type TickResult struct {
	BroadcastRoomIDs  []string
	ScheduleAIRoomIDs []string
	DestroyedRoomIDs  []string
}

func NewManager(aiProvider aiplayer.Provider) *Manager {
	return &Manager{
		RoomRuntime:  gameactor.NewRoomRuntime(64),
		aiProvider:   aiProvider,
		rooms:        map[string]*Room{},
		aiController: aiagent.NewController("uno", aiProvider, aiplayer.DecisionTimeout),
	}
}
