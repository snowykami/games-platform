package mahjong

import (
	"sync"

	"github.com/snowykami/games-platform/server/internal/aiagent"
	"github.com/snowykami/games-platform/server/internal/aiplayer"
	"github.com/snowykami/games-platform/server/internal/gameactor"
)

const (
	minPlayers = 4
	maxPlayers = 4
)

var (
	winds = []Wind{WindEast, WindSouth, WindWest, WindNorth}
	rules = RuleSet{
		ID:          "chinese-official",
		Name:        "国标麻将",
		MinFan:      8,
		Description: "首版实现国标 8 番起胡与常用番型子集，规则层可继续扩展。",
	}
)

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
		aiController: aiagent.NewController("mahjong", aiProvider, aiplayer.DecisionTimeout),
	}
}
