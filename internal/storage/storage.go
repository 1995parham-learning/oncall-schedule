package storage

import (
	"sync"
	"time"
)

// Team represents a team with their schedules.
type Team struct {
	Schedules []Schedule
}

// Schedule represents an on-call schedule.
type Schedule struct {
	Name    string
	Members []string
	Days    []time.Weekday
	Start   time.Time
	End     time.Time
}

// Storage defines the interface for storing and retrieving schedules.
type Storage interface {
	AddSchedule(team string, schedule Schedule)
	GetTeam(team string) (Team, bool)
}

// MemoryStorage implements Storage interface with thread-safe in-memory storage.
type MemoryStorage struct {
	mu   sync.RWMutex
	data map[string]Team
}

// NewMemoryStorage creates a new memory storage instance.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]Team),
	}
}

// AddSchedule adds a schedule to a team (thread-safe).
func (s *MemoryStorage) AddSchedule(team string, schedule Schedule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.data[team]
	t.Schedules = append(t.Schedules, schedule)
	s.data[team] = t
}

// GetTeam retrieves a team's schedules (thread-safe).
func (s *MemoryStorage) GetTeam(team string) (Team, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.data[team]
	return t, ok
}
