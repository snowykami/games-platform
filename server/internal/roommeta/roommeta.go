package roommeta

import (
	"errors"
	"strings"
)

const (
	maxDisplayNameRunes = 24
	maxNoteRunes        = 80
)

func NormalizeDisplayName(value string) (string, error) {
	name := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if name == "" {
		return "", errors.New("invalid_display_name")
	}
	runes := []rune(name)
	if len(runes) > maxDisplayNameRunes {
		name = string(runes[:maxDisplayNameRunes])
	}
	return name, nil
}

func NormalizeNote(value string) string {
	note := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	runes := []rune(note)
	if len(runes) > maxNoteRunes {
		return string(runes[:maxNoteRunes])
	}
	return note
}
