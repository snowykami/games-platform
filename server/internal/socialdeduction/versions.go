package socialdeduction

import "time"

func touchRule(room *Room) {
	now := time.Now().UTC()
	room.RuleUpdatedAt = now
	room.UpdatedAt = now
}

func touchSpeech(room *Room) {
	now := time.Now().UTC()
	if room.RuleUpdatedAt.IsZero() {
		room.RuleUpdatedAt = room.UpdatedAt
	}
	room.SpeechUpdatedAt = now
	room.UpdatedAt = now
}

func touchRuleAndSpeech(room *Room) {
	now := time.Now().UTC()
	room.RuleUpdatedAt = now
	room.SpeechUpdatedAt = now
	room.UpdatedAt = now
}

func touchPresence(room *Room) {
	now := time.Now().UTC()
	if room.RuleUpdatedAt.IsZero() {
		room.RuleUpdatedAt = room.UpdatedAt
	}
	room.PresenceUpdatedAt = now
	room.UpdatedAt = now
}

func touchView(room *Room) {
	if room.RuleUpdatedAt.IsZero() {
		room.RuleUpdatedAt = room.UpdatedAt
	}
	room.UpdatedAt = time.Now().UTC()
}

func decisionRuleUpdatedAt(room *Room) time.Time {
	return room.RuleUpdatedAt
}

func decisionSpeechUpdatedAt(room *Room) time.Time {
	return room.SpeechUpdatedAt
}
