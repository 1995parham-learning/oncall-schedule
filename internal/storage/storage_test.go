package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_AddSchedule(t *testing.T) {
	storage := NewMemoryStorage()

	schedule := Schedule{
		Name:    "Weekend Coverage",
		Members: []string{"Alice", "Bob", "Charlie"},
		Days:    []time.Weekday{time.Saturday, time.Sunday},
		Start:   parseTime(t, "9:00AM"),
		End:     parseTime(t, "5:00PM"),
	}

	err := storage.AddSchedule("backend-team", schedule)
	require.NoError(t, err)

	// Verify the schedule was added
	team, ok, err := storage.GetTeam("backend-team")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, team.Schedules, 1)
	assert.Equal(t, "Weekend Coverage", team.Schedules[0].Name)
	assert.Equal(t, []string{"Alice", "Bob", "Charlie"}, team.Schedules[0].Members)
}

func TestMemoryStorage_AddMultipleSchedules(t *testing.T) {
	storage := NewMemoryStorage()

	schedule1 := Schedule{
		Name:    "Weekday Morning",
		Members: []string{"Alice", "Bob"},
		Days:    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:   parseTime(t, "9:00AM"),
		End:     parseTime(t, "5:00PM"),
	}

	schedule2 := Schedule{
		Name:    "Weekday Evening",
		Members: []string{"Charlie", "David"},
		Days:    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:   parseTime(t, "5:00PM"),
		End:     parseTime(t, "11:00PM"),
	}

	err := storage.AddSchedule("backend-team", schedule1)
	require.NoError(t, err)

	err = storage.AddSchedule("backend-team", schedule2)
	require.NoError(t, err)

	// Verify both schedules exist
	team, ok, err := storage.GetTeam("backend-team")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, team.Schedules, 2)
}

func TestMemoryStorage_GetTeam_NotFound(t *testing.T) {
	storage := NewMemoryStorage()

	team, ok, err := storage.GetTeam("non-existent-team")
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, team.Schedules)
}

func TestMemoryStorage_GetCurrentOncall(t *testing.T) {
	storage := NewMemoryStorage()

	schedule := Schedule{
		Name:    "Weekday Coverage",
		Members: []string{"Alice", "Bob", "Charlie"},
		Days:    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:   parseTime(t, "9:00AM"),
		End:     parseTime(t, "5:00PM"),
	}

	err := storage.AddSchedule("backend-team", schedule)
	require.NoError(t, err)

	tests := []struct {
		name        string
		queryTime   time.Time
		expectedOk  bool
		expectedMember string
	}{
		{
			name:        "During schedule - Monday morning",
			queryTime:   time.Date(2025, 4, 28, 10, 0, 0, 0, time.UTC), // Monday 10:00 AM
			expectedOk:  true,
			expectedMember: "Alice", // First member in rotation
		},
		{
			name:        "During schedule - Friday afternoon",
			queryTime:   time.Date(2025, 5, 2, 14, 0, 0, 0, time.UTC), // Friday 2:00 PM
			expectedOk:  true,
			expectedMember: "Alice",
		},
		{
			name:       "Outside schedule - Saturday",
			queryTime:  time.Date(2025, 4, 26, 10, 0, 0, 0, time.UTC), // Saturday 10:00 AM
			expectedOk: false,
		},
		{
			name:       "Outside schedule - too early",
			queryTime:  time.Date(2025, 4, 28, 8, 0, 0, 0, time.UTC), // Monday 8:00 AM
			expectedOk: false,
		},
		{
			name:       "Outside schedule - too late",
			queryTime:  time.Date(2025, 4, 28, 18, 0, 0, 0, time.UTC), // Monday 6:00 PM
			expectedOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oncall, ok, err := storage.GetCurrentOncall("backend-team", tt.queryTime)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOk, ok)
			if tt.expectedOk {
				assert.Equal(t, tt.expectedMember, oncall)
			}
		})
	}
}

func TestMemoryStorage_GetCurrentOncall_TeamNotFound(t *testing.T) {
	storage := NewMemoryStorage()

	oncall, ok, err := storage.GetCurrentOncall("non-existent-team", time.Now())
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, oncall)
}

func TestMemoryStorage_GetCurrentOncall_EmptyMembers(t *testing.T) {
	storage := NewMemoryStorage()

	schedule := Schedule{
		Name:    "Empty Schedule",
		Members: []string{}, // Empty members list
		Days:    []time.Weekday{time.Monday},
		Start:   parseTime(t, "9:00AM"),
		End:     parseTime(t, "5:00PM"),
	}

	err := storage.AddSchedule("backend-team", schedule)
	require.NoError(t, err)

	queryTime := time.Date(2025, 4, 28, 10, 0, 0, 0, time.UTC) // Monday 10:00 AM
	oncall, ok, err := storage.GetCurrentOncall("backend-team", queryTime)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Empty(t, oncall)
}

func TestMemoryStorage_ThreadSafety(t *testing.T) {
	storage := NewMemoryStorage()

	// Spawn multiple goroutines to test thread safety
	done := make(chan bool)

	// Writers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			schedule := Schedule{
				Name:    "Schedule",
				Members: []string{"Alice"},
				Days:    []time.Weekday{time.Monday},
				Start:   parseTime(t, "9:00AM"),
				End:     parseTime(t, "5:00PM"),
			}
			_ = storage.AddSchedule("team", schedule)
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		go func() {
			_, _, _ = storage.GetTeam("team")
			done <- true
		}()
	}

	// Oncall readers
	for i := 0; i < 10; i++ {
		go func() {
			_, _, _ = storage.GetCurrentOncall("team", time.Now())
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 30; i++ {
		<-done
	}
}

// parseTime is a helper function to parse time strings in tests
func parseTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.Kitchen, timeStr)
	require.NoError(t, err)
	return parsed
}
