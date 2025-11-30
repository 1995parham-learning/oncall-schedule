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
	AddSchedule(team string, schedule Schedule) error
	GetTeam(team string) (Team, bool, error)
	GetCurrentOncall(team string, at time.Time) (string, bool, error)
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
func (s *MemoryStorage) AddSchedule(team string, schedule Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.data[team]
	t.Schedules = append(t.Schedules, schedule)
	s.data[team] = t
	return nil
}

// GetTeam retrieves a team's schedules (thread-safe).
func (s *MemoryStorage) GetTeam(team string) (Team, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.data[team]
	return t, ok, nil
}

// GetCurrentOncall returns the first member of the first matching schedule.
// Note: This is a simplified implementation for in-memory storage.
// It doesn't implement proper rotation tracking.
func (s *MemoryStorage) GetCurrentOncall(team string, at time.Time) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.data[team]
	if !ok {
		return "", false, nil
	}

	// Check each schedule to find a match
	for _, sched := range t.Schedules {
		// Check if day matches
		dayMatches := false
		for _, day := range sched.Days {
			if day == at.Weekday() {
				dayMatches = true
				break
			}
		}
		if !dayMatches {
			continue
		}

		// Check if time is within schedule
		schedTime := time.Date(at.Year(), at.Month(), at.Day(),
			at.Hour(), at.Minute(), at.Second(), at.Nanosecond(), at.Location())
		schedStart := time.Date(at.Year(), at.Month(), at.Day(),
			sched.Start.Hour(), sched.Start.Minute(), sched.Start.Second(), 0, at.Location())
		schedEnd := time.Date(at.Year(), at.Month(), at.Day(),
			sched.End.Hour(), sched.End.Minute(), sched.End.Second(), 0, at.Location())

		if schedTime.After(schedStart) && schedTime.Before(schedEnd) || schedTime.Equal(schedStart) {
			if len(sched.Members) > 0 {
				// Return first member (no rotation tracking in memory storage)
				return sched.Members[0], true, nil
			}
		}
	}

	return "", false, nil
}
