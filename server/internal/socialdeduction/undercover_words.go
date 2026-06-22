package socialdeduction

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
)

const minUndercoverGroupsPerDomain = 100

//go:embed data/undercover/*.json
var undercoverWordBankFS embed.FS

type undercoverDomainSource struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Groups      [][]undercoverWordEntry `json:"groups"`
}

type undercoverWordEntry struct {
	Text string `json:"text"`
	Hint string `json:"hint,omitempty"`
}

type undercoverWordGroup struct {
	DomainID   string
	Category   string
	GroupIndex int
	Words      []undercoverWordEntry
}

func (entry *undercoverWordEntry) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		entry.Text = text
		entry.Hint = ""
		return nil
	}
	type wordObject undercoverWordEntry
	var object wordObject
	if err := json.Unmarshal(data, &object); err != nil {
		return err
	}
	entry.Text = object.Text
	entry.Hint = object.Hint
	return nil
}

var undercoverWordBankCache = struct {
	sync.Once
	sources []undercoverDomainSource
	err     error
}{}

func undercoverDomainSources() []undercoverDomainSource {
	undercoverWordBankCache.Do(func() {
		undercoverWordBankCache.sources, undercoverWordBankCache.err = loadUndercoverDomainSources()
	})
	if undercoverWordBankCache.err != nil {
		panic(undercoverWordBankCache.err)
	}
	return undercoverWordBankCache.sources
}

func loadUndercoverDomainSources() ([]undercoverDomainSource, error) {
	paths, err := fs.Glob(undercoverWordBankFS, "data/undercover/*.json")
	if err != nil {
		return nil, fmt.Errorf("glob undercover word bank: %w", err)
	}
	slices.Sort(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("undercover word bank is empty")
	}

	sources := make([]undercoverDomainSource, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		source, err := readUndercoverDomainSource(path)
		if err != nil {
			return nil, err
		}
		if seen[source.ID] {
			return nil, fmt.Errorf("duplicate undercover domain %q", source.ID)
		}
		seen[source.ID] = true
		sources = append(sources, source)
	}
	if !seen[defaultUndercoverPresetID()] {
		return nil, fmt.Errorf("default undercover domain %q is missing", defaultUndercoverPresetID())
	}
	return sources, nil
}

func readUndercoverDomainSource(path string) (undercoverDomainSource, error) {
	data, err := undercoverWordBankFS.ReadFile(path)
	if err != nil {
		return undercoverDomainSource{}, fmt.Errorf("read undercover word bank %s: %w", path, err)
	}
	var source undercoverDomainSource
	if err := json.Unmarshal(data, &source); err != nil {
		return undercoverDomainSource{}, fmt.Errorf("decode undercover word bank %s: %w", path, err)
	}
	if err := normalizeUndercoverDomainSource(&source); err != nil {
		return undercoverDomainSource{}, fmt.Errorf("invalid undercover word bank %s: %w", path, err)
	}
	return source, nil
}

func normalizeUndercoverDomainSource(source *undercoverDomainSource) error {
	source.ID = strings.TrimSpace(source.ID)
	source.Name = strings.TrimSpace(source.Name)
	source.Description = strings.TrimSpace(source.Description)
	if source.ID == "" || source.Name == "" || source.Description == "" {
		return fmt.Errorf("id, name and description are required")
	}
	if len(source.Groups) < minUndercoverGroupsPerDomain {
		return fmt.Errorf("domain %q has %d groups, expected at least %d", source.ID, len(source.Groups), minUndercoverGroupsPerDomain)
	}

	seenGroups := map[string]bool{}
	groups := make([][]undercoverWordEntry, 0, len(source.Groups))
	for index, group := range source.Groups {
		if len(group) < 2 {
			return fmt.Errorf("domain %q group %d must contain at least two words", source.ID, index+1)
		}
		wordSeen := map[string]bool{}
		words := make([]undercoverWordEntry, 0, len(group))
		for wordIndex, rawEntry := range group {
			word := strings.TrimSpace(rawEntry.Text)
			if word == "" {
				return fmt.Errorf("domain %q group %d word %d is empty", source.ID, index+1, wordIndex+1)
			}
			key := strings.ToLower(word)
			if wordSeen[key] {
				return fmt.Errorf("domain %q group %d repeats word %q", source.ID, index+1, word)
			}
			wordSeen[key] = true
			words = append(words, undercoverWordEntry{
				Text: word,
				Hint: strings.TrimSpace(rawEntry.Hint),
			})
		}
		groupKey := normalizedUndercoverGroupKey(words)
		if seenGroups[groupKey] {
			return fmt.Errorf("domain %q group %d duplicates word group", source.ID, index+1)
		}
		seenGroups[groupKey] = true
		groups = append(groups, words)
	}
	source.Groups = groups
	return nil
}

func normalizedUndercoverGroupKey(words []undercoverWordEntry) string {
	parts := make([]string, 0, len(words))
	for _, word := range words {
		parts = append(parts, strings.ToLower(word.Text))
	}
	return strings.Join(parts, "\x00")
}
