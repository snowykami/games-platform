package mahjong

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
)

func createWall() []Tile {
	wall := []Tile{}
	for _, suit := range []struct {
		prefix string
		kind   TileKind
	}{{"m", TileCharacters}, {"p", TileDots}, {"s", TileBamboo}} {
		for rank := 1; rank <= 9; rank++ {
			for copyIndex := 0; copyIndex < 4; copyIndex++ {
				wall = append(wall, Tile{ID: fmt.Sprintf("%s%d_%d_%s", suit.prefix, rank, copyIndex, randomToken(6)), Code: fmt.Sprintf("%s%d", suit.prefix, rank), Kind: suit.kind, Rank: rank})
			}
		}
	}
	for _, wind := range winds {
		for copyIndex := 0; copyIndex < 4; copyIndex++ {
			code := windCode(wind)
			wall = append(wall, Tile{ID: fmt.Sprintf("%s_%d_%s", code, copyIndex, randomToken(6)), Code: code, Kind: TileWind, Wind: wind})
		}
	}
	for _, dragon := range []Dragon{DragonRed, DragonGreen, DragonWhite} {
		for copyIndex := 0; copyIndex < 4; copyIndex++ {
			code := dragonCode(dragon)
			wall = append(wall, Tile{ID: fmt.Sprintf("%s_%d_%s", code, copyIndex, randomToken(6)), Code: code, Kind: TileDragon, Dragon: dragon})
		}
	}
	return wall
}

func shuffle(tiles []Tile) []Tile {
	shuffled := append([]Tile(nil), tiles...)
	rand.Shuffle(len(shuffled), func(i int, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func shiftTiles(tiles *[]Tile, count int) []Tile {
	if count > len(*tiles) {
		count = len(*tiles)
	}
	shifted := append([]Tile{}, (*tiles)[:count]...)
	*tiles = (*tiles)[count:]
	return shifted
}

func sortTiles(tiles []Tile) []Tile {
	sorted := append([]Tile{}, tiles...)
	sort.Slice(sorted, func(i int, j int) bool { return codeOrder(sorted[i].Code) < codeOrder(sorted[j].Code) })
	return sorted
}

func filterTiles(tiles []Tile, keep func(Tile) bool) []Tile {
	filtered := []Tile{}
	for _, tile := range tiles {
		if keep(tile) {
			filtered = append(filtered, tile)
		}
	}
	return filtered
}

func firstTileByCode(tiles []Tile, code string) (Tile, bool) {
	for _, tile := range tiles {
		if tile.Code == code {
			return tile, true
		}
	}
	return Tile{}, false
}

func tileCodes(tiles []Tile) []string {
	codes := []string{}
	for _, tile := range tiles {
		codes = append(codes, tile.Code)
	}
	sort.Slice(codes, func(i int, j int) bool { return codeOrder(codes[i]) < codeOrder(codes[j]) })
	return codes
}

func countCodes(codes []string) map[string]int {
	counts := map[string]int{}
	for _, code := range codes {
		counts[code]++
	}
	return counts
}

func cloneCounts(counts map[string]int) map[string]int {
	next := map[string]int{}
	for code, count := range counts {
		next[code] = count
	}
	return next
}

func totalCount(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		if count > 0 {
			total += count
		}
	}
	return total
}

func firstRemainingCode(counts map[string]int) string {
	codes := []string{}
	for code, count := range counts {
		if count > 0 {
			codes = append(codes, code)
		}
	}
	sort.Slice(codes, func(i int, j int) bool { return codeOrder(codes[i]) < codeOrder(codes[j]) })
	if len(codes) == 0 {
		return ""
	}
	return codes[0]
}

func chowCodes(code string) []string {
	if !isSuited(code) || len(code) < 2 {
		return nil
	}
	rank := int(code[1] - '0')
	if rank > 7 {
		return nil
	}
	return []string{fmt.Sprintf("%s%d", code[:1], rank), fmt.Sprintf("%s%d", code[:1], rank+1), fmt.Sprintf("%s%d", code[:1], rank+2)}
}

func isSuited(code string) bool {
	return strings.HasPrefix(code, "m") || strings.HasPrefix(code, "p") || strings.HasPrefix(code, "s")
}

func isHonorCode(code string) bool {
	return strings.HasPrefix(code, "z")
}

func isDragonCode(code string) bool {
	return code == "z5" || code == "z6" || code == "z7"
}

func windCode(wind Wind) string {
	switch wind {
	case WindEast:
		return "z1"
	case WindSouth:
		return "z2"
	case WindWest:
		return "z3"
	case WindNorth:
		return "z4"
	default:
		return "z1"
	}
}

func dragonCode(dragon Dragon) string {
	switch dragon {
	case DragonRed:
		return "z5"
	case DragonGreen:
		return "z6"
	case DragonWhite:
		return "z7"
	default:
		return "z7"
	}
}

func codeOrder(code string) int {
	base := map[byte]int{'m': 0, 'p': 20, 's': 40, 'z': 60}[code[0]]
	return base + int(code[1]-'0')
}

func formatTile(tile Tile) string {
	if tile.Rank > 0 {
		switch tile.Kind {
		case TileCharacters:
			return fmt.Sprintf("%d万", tile.Rank)
		case TileDots:
			return fmt.Sprintf("%d筒", tile.Rank)
		case TileBamboo:
			return fmt.Sprintf("%d条", tile.Rank)
		}
	}
	switch tile.Code {
	case "z1":
		return "东"
	case "z2":
		return "南"
	case "z3":
		return "西"
	case "z4":
		return "北"
	case "z5":
		return "中"
	case "z6":
		return "发"
	default:
		return "白"
	}
}
