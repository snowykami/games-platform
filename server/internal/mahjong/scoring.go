package mahjong

import (
	"fmt"
	"slices"
)

func evaluateWin(tiles []Tile, melds []Meld, selfDraw bool, seatWind Wind, roundWind Wind, ruleset RuleSet) WinResult {
	codes := tileCodes(tiles)
	if len(melds) == 0 && isSevenPairs(codes) {
		result := WinResult{Fan: 24, Patterns: []FanPattern{{Name: "七对", Fan: 24}}}
		if selfDraw {
			result.Fan++
			result.Patterns = append(result.Patterns, FanPattern{Name: "自摸", Fan: 1})
		}
		result.CanWin = result.Fan >= ruleset.MinFan
		return result
	}
	neededMelds := 4 - len(melds)
	if (len(codes)-2)/3 != neededMelds || (len(codes)-2)%3 != 0 {
		return WinResult{Reason: "牌型还没有组成 4 副面子加 1 对将。"}
	}
	decompositions := decompose(codes, neededMelds)
	if len(decompositions) == 0 {
		return WinResult{Reason: "牌型还没有组成 4 副面子加 1 对将。"}
	}
	best := WinResult{}
	for _, meldCodes := range decompositions {
		result := scoreWin(codes, meldCodes, melds, selfDraw, seatWind, roundWind)
		if result.Fan > best.Fan {
			best = result
		}
	}
	if best.Fan < ruleset.MinFan {
		best.CanWin = false
		best.Reason = fmt.Sprintf("国标麻将至少 %d 番起胡，当前 %d 番。", ruleset.MinFan, best.Fan)
		return best
	}
	best.CanWin = true
	return best
}

type codedMeld struct {
	kind  MeldKind
	codes []string
}

func scoreWin(codes []string, concealedMelds []codedMeld, exposedMelds []Meld, selfDraw bool, seatWind Wind, roundWind Wind) WinResult {
	patterns := []FanPattern{}
	allMelds := append([]codedMeld{}, concealedMelds...)
	for _, meld := range exposedMelds {
		allMelds = append(allMelds, codedMeld{kind: meld.Kind, codes: tileCodes(meld.Tiles)})
	}
	allCodes := append([]string{}, codes...)
	for _, meld := range exposedMelds {
		allCodes = append(allCodes, tileCodes(meld.Tiles)...)
	}
	if isPureOneSuit(allCodes) {
		patterns = append(patterns, FanPattern{Name: "清一色", Fan: 24})
	} else if isHalfFlush(allCodes) {
		patterns = append(patterns, FanPattern{Name: "混一色", Fan: 6})
	}
	if everyMeld(allMelds, func(meld codedMeld) bool { return meld.kind == MeldPung || meld.kind == MeldKong }) {
		patterns = append(patterns, FanPattern{Name: "碰碰和", Fan: 6})
	}
	dragonPungs := 0
	for _, meld := range allMelds {
		if (meld.kind == MeldPung || meld.kind == MeldKong) && isDragonCode(meld.codes[0]) {
			dragonPungs++
		}
	}
	for range dragonPungs {
		patterns = append(patterns, FanPattern{Name: "箭刻", Fan: 2})
	}
	if hasPung(allMelds, windCode(seatWind)) {
		patterns = append(patterns, FanPattern{Name: "门风刻", Fan: 2})
	}
	if hasPung(allMelds, windCode(roundWind)) {
		patterns = append(patterns, FanPattern{Name: "圈风刻", Fan: 2})
	}
	if selfDraw {
		patterns = append(patterns, FanPattern{Name: "自摸", Fan: 1})
	}
	if !slices.ContainsFunc(allCodes, isHonorCode) {
		patterns = append(patterns, FanPattern{Name: "无字", Fan: 1})
	}
	fan := 0
	for _, pattern := range patterns {
		fan += pattern.Fan
	}
	return WinResult{Fan: fan, Patterns: patterns}
}

func decompose(codes []string, neededMelds int) [][]codedMeld {
	counts := countCodes(codes)
	results := [][]codedMeld{}
	for code, count := range counts {
		if count < 2 {
			continue
		}
		nextCounts := cloneCounts(counts)
		nextCounts[code] -= 2
		for _, melds := range findMelds(nextCounts, neededMelds) {
			results = append(results, melds)
		}
	}
	return results
}

func findMelds(counts map[string]int, remaining int) [][]codedMeld {
	if remaining == 0 {
		if totalCount(counts) == 0 {
			return [][]codedMeld{{}}
		}
		return nil
	}
	code := firstRemainingCode(counts)
	if code == "" {
		return nil
	}
	results := [][]codedMeld{}
	if counts[code] >= 3 {
		nextCounts := cloneCounts(counts)
		nextCounts[code] -= 3
		for _, melds := range findMelds(nextCounts, remaining-1) {
			results = append(results, append([]codedMeld{{kind: MeldPung, codes: []string{code, code, code}}}, melds...))
		}
	}
	chowCodes := chowCodes(code)
	if len(chowCodes) == 3 && counts[chowCodes[0]] > 0 && counts[chowCodes[1]] > 0 && counts[chowCodes[2]] > 0 {
		nextCounts := cloneCounts(counts)
		for _, chowCode := range chowCodes {
			nextCounts[chowCode]--
		}
		for _, melds := range findMelds(nextCounts, remaining-1) {
			results = append(results, append([]codedMeld{{kind: MeldChow, codes: chowCodes}}, melds...))
		}
	}
	return results
}

func isSevenPairs(codes []string) bool {
	if len(codes) != 14 {
		return false
	}
	pairs := 0
	for _, count := range countCodes(codes) {
		if count == 2 {
			pairs++
		}
		if count == 4 {
			pairs += 2
		}
	}
	return pairs == 7
}

func isPureOneSuit(codes []string) bool {
	suit := ""
	for _, code := range codes {
		if isHonorCode(code) {
			return false
		}
		if suit == "" {
			suit = code[:1]
		}
		if suit != code[:1] {
			return false
		}
	}
	return suit != ""
}

func isHalfFlush(codes []string) bool {
	suit := ""
	hasHonor := false
	for _, code := range codes {
		if isHonorCode(code) {
			hasHonor = true
			continue
		}
		if suit == "" {
			suit = code[:1]
		}
		if suit != code[:1] {
			return false
		}
	}
	return suit != "" && hasHonor
}

func hasPung(melds []codedMeld, code string) bool {
	for _, meld := range melds {
		if (meld.kind == MeldPung || meld.kind == MeldKong) && meld.codes[0] == code {
			return true
		}
	}
	return false
}

func everyMeld(melds []codedMeld, keep func(codedMeld) bool) bool {
	if len(melds) == 0 {
		return false
	}
	for _, meld := range melds {
		if !keep(meld) {
			return false
		}
	}
	return true
}
