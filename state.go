package main

import (
	"encoding/json"
	"os"
	"time"
)

// seenStore remembers which posts were already reported so reruns only emit new ones.
const seenFile = "output/seen.json"

type seenStore struct {
	Seen map[string]string `json:"seen"` // dedupe key -> first seen (RFC3339)
}

func loadSeen() *seenStore {
	s := &seenStore{Seen: map[string]string{}}
	data, err := os.ReadFile(seenFile)
	if err == nil {
		_ = json.Unmarshal(data, s)
		if s.Seen == nil {
			s.Seen = map[string]string{}
		}
	}
	return s
}

func (s *seenStore) IsNew(p Post) bool {
	_, ok := s.Seen[dedupeKey(p)]
	return !ok
}

func (s *seenStore) Mark(p Post) {
	s.Seen[dedupeKey(p)] = time.Now().UTC().Format(time.RFC3339)
}

func (s *seenStore) Save() error {
	return writeJSON(seenFile, s)
}
