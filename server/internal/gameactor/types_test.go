package gameactor

import (
	"testing"

	"github.com/snowykami/games-platform/server/internal/aiplayer"
)

func TestRoomVersionSeparatesRuleSpeechAndPresence(t *testing.T) {
	version := RoomVersion{}

	afterSpeech := version.Bump(LaneSpeech)
	if afterSpeech.RuleVersion != 0 || afterSpeech.SpeechVersion != 1 || afterSpeech.ViewVersion != 1 {
		t.Fatalf("speech bump = %+v, want only speech and view changed", afterSpeech)
	}

	afterPresence := afterSpeech.Bump(LanePresence)
	if afterPresence.RuleVersion != 0 || afterPresence.PresenceVersion != 1 || afterPresence.ViewVersion != 2 {
		t.Fatalf("presence bump = %+v, want only presence and view changed", afterPresence)
	}

	afterRule := afterPresence.Bump(LaneRule)
	if afterRule.RuleVersion != 1 || afterRule.SpeechVersion != 1 || afterRule.PresenceVersion != 1 || afterRule.ViewVersion != 3 {
		t.Fatalf("rule bump = %+v, want rule and view changed", afterRule)
	}
}

func TestRequiredActionDoesNotBecomeStaleFromSpeechVersion(t *testing.T) {
	actions := []aiplayer.LegalAction{{ID: "vote:seat_2", Label: "投票给座位2"}}
	start := RoomVersion{RuleVersion: 7, SpeechVersion: 2}
	request := ActiveAgentRequest{
		PlayerID:        "ai_1",
		Kind:            AgentRequiredAction,
		Phase:           "vote",
		RuleVersion:     start.RuleVersion,
		LegalActionHash: LegalActionHash(actions),
	}

	current := start.Bump(LaneSpeech)
	if reason := request.StaleReason(current, "vote", "ai_1", actions); reason != "" {
		t.Fatalf("speech update made required action stale: %s", reason)
	}
}

func TestRequiredActionBecomesStaleFromRuleVersionOrLegalActionChange(t *testing.T) {
	actions := []aiplayer.LegalAction{{ID: "vote:seat_2", Label: "投票给座位2"}}
	start := RoomVersion{RuleVersion: 7}
	request := ActiveAgentRequest{
		PlayerID:        "ai_1",
		Kind:            AgentRequiredAction,
		Phase:           "vote",
		RuleVersion:     start.RuleVersion,
		LegalActionHash: LegalActionHash(actions),
	}

	if reason := request.StaleReason(start.Bump(LaneRule), "vote", "ai_1", actions); reason != "rule_version_changed" {
		t.Fatalf("stale reason = %q, want rule_version_changed", reason)
	}

	changedActions := []aiplayer.LegalAction{{ID: "vote:seat_3", Label: "投票给座位3"}}
	if reason := request.StaleReason(start, "vote", "ai_1", changedActions); reason != "legal_actions_changed" {
		t.Fatalf("stale reason = %q, want legal_actions_changed", reason)
	}
}

func TestOptionalSpeechBecomesStaleFromSpeechVersion(t *testing.T) {
	start := RoomVersion{SpeechVersion: 3}
	request := ActiveAgentRequest{
		PlayerID:      "ai_1",
		Kind:          AgentOptionalSpeech,
		Phase:         "day",
		SpeechVersion: start.SpeechVersion,
	}

	if reason := request.StaleReason(start.Bump(LaneSpeech), "day", "ai_1", nil); reason != "speech_version_changed" {
		t.Fatalf("stale reason = %q, want speech_version_changed", reason)
	}
}
