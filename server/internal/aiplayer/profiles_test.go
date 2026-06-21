package aiplayer

import "testing"

func TestProfilesExposeSharedPool(t *testing.T) {
	profiles := Profiles()
	if len(profiles) < 10 {
		t.Fatalf("profiles length = %d, want at least 10", len(profiles))
	}
	for _, profile := range profiles {
		if profile.Name == "" || profile.Personality == "" || profile.SpeechStyle == "" {
			t.Fatalf("profile must include name, personality, and speech style: %+v", profile)
		}
	}
	if !hasProfileName(profiles, "Harper") || !hasProfileName(profiles, "Sora") || !hasProfileName(profiles, "さくら") || !hasProfileName(profiles, "林澈") {
		t.Fatalf("expected shared pool to include mixed English, romaji, Japanese, and Chinese names, got %+v", profiles)
	}
}

func hasProfileName(profiles []BotProfile, name string) bool {
	for _, profile := range profiles {
		if profile.Name == name {
			return true
		}
	}
	return false
}
