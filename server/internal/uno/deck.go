package uno

import "math/rand/v2"

func createDeck(rules RuleSet) []Card {
	cards := []Card{}
	if rules.AllWild {
		for range 18 {
			cards = append(cards, createCard(ColorWild, KindWild))
		}
		for range 8 {
			cards = append(cards, createCard(ColorWild, KindSkip), createCard(ColorWild, KindReverse), createCard(ColorWild, KindDrawTwo))
		}
		for range 4 {
			cards = append(cards, createCard(ColorWild, KindWildDrawFour), createCard(ColorWild, KindWildDrawSix))
		}
		return cards
	}
	for _, color := range colors {
		for value := range 10 {
			cards = append(cards, createNumberCard(color, value))
			if value != 0 {
				cards = append(cards, createNumberCard(color, value))
			}
		}
		for range 2 {
			cards = append(cards, createCard(color, KindSkip), createCard(color, KindReverse), createCard(color, KindDrawTwo))
		}
	}
	for range 4 {
		cards = append(cards, createCard(ColorWild, KindWild), createCard(ColorWild, KindWildDrawFour))
	}
	if rules.Flip {
		for range 8 {
			cards = append(cards, createCard(ColorWild, KindFlip))
		}
	}
	if rules.NoMercy {
		for range 4 {
			cards = append(cards, createCard(ColorWild, KindWildDrawSix), createCard(ColorWild, KindWildDrawTen))
		}
	}
	return cards
}

func createNumberCard(color Color, value int) Card {
	return Card{ID: "card_" + randomToken(10), Color: color, Kind: KindNumber, Value: &value}
}

func createCard(color Color, kind Kind) Card {
	return Card{ID: "card_" + randomToken(10), Color: color, Kind: kind}
}

func shuffle(cards []Card) []Card {
	shuffled := append([]Card(nil), cards...)
	rand.Shuffle(len(shuffled), func(i int, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func chooseBestColor(hand []Card) Color {
	score := map[Color]int{}
	for _, card := range hand {
		if isRealColor(card.Color) {
			score[card.Color]++
		}
	}

	best := ColorRed
	for _, color := range colors {
		if score[color] > score[best] {
			best = color
		}
	}
	return best
}

func startingColor(card Card) Color {
	if isRealColor(card.Color) {
		return card.Color
	}
	return ColorRed
}

func sameFace(a Card, b Card) bool {
	if a.Color != b.Color || a.Kind != b.Kind {
		return false
	}
	if a.Kind != KindNumber {
		return true
	}
	if a.Value == nil || b.Value == nil {
		return false
	}
	return *a.Value == *b.Value
}

func drawPenalty(kind Kind) int {
	switch kind {
	case KindDrawTwo:
		return 2
	case KindWildDrawFour:
		return 4
	case KindWildDrawSix:
		return 6
	case KindWildDrawTen:
		return 10
	default:
		return 0
	}
}

func isRealColor(color Color) bool {
	return color == ColorRed || color == ColorYellow || color == ColorGreen || color == ColorBlue
}
