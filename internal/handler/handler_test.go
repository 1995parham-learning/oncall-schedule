package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/1995parham-learning/oncall-schedule/internal/storage"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateSchedule_Success(t *testing.T) {
	// Setup
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	reqBody := Request{
		Name:    "Weekday Coverage",
		Team:    "backend-team",
		Members: []string{"Alice", "Bob", "Charlie"},
		Days:    []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
		Start:   "9:00AM",
		End:     "5:00PM",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.CreateSchedule(c)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Verify schedule was created
	team, ok, err := store.GetTeam("backend-team")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Len(t, team.Schedules, 1)
	assert.Equal(t, "Weekday Coverage", team.Schedules[0].Name)
}

func TestCreateSchedule_InvalidJSON(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader([]byte("invalid json")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.CreateSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid request body")
}

func TestCreateSchedule_MissingFields(t *testing.T) {
	tests := []struct {
		name        string
		req         Request
		expectedErr string
	}{
		{
			name: "missing team",
			req: Request{
				Name:    "Schedule",
				Members: []string{"Alice"},
				Days:    []string{"Monday"},
				Start:   "9:00AM",
				End:     "5:00PM",
			},
			expectedErr: "team is required",
		},
		{
			name: "missing members",
			req: Request{
				Name:    "Schedule",
				Team:    "team",
				Members: []string{},
				Days:    []string{"Monday"},
				Start:   "9:00AM",
				End:     "5:00PM",
			},
			expectedErr: "at least one member is required",
		},
		{
			name: "missing days",
			req: Request{
				Name:    "Schedule",
				Team:    "team",
				Members: []string{"Alice"},
				Days:    []string{},
				Start:   "9:00AM",
				End:     "5:00PM",
			},
			expectedErr: "at least one day is required",
		},
		{
			name: "missing start time",
			req: Request{
				Name:    "Schedule",
				Team:    "team",
				Members: []string{"Alice"},
				Days:    []string{"Monday"},
				End:     "5:00PM",
			},
			expectedErr: "start time is required",
		},
		{
			name: "missing end time",
			req: Request{
				Name:    "Schedule",
				Team:    "team",
				Members: []string{"Alice"},
				Days:    []string{"Monday"},
				Start:   "9:00AM",
			},
			expectedErr: "end time is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			store := storage.NewMemoryStorage()
			logger, _ := zap.NewDevelopment()
			h := New(store, logger)

			body, err := json.Marshal(tt.req)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err = h.CreateSchedule(c)

			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &errResp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedErr, errResp.Error)
		})
	}
}

func TestCreateSchedule_InvalidDay(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	reqBody := Request{
		Name:    "Schedule",
		Team:    "team",
		Members: []string{"Alice"},
		Days:    []string{"InvalidDay"},
		Start:   "9:00AM",
		End:     "5:00PM",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.CreateSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid day")
}

func TestCreateSchedule_InvalidTimeFormat(t *testing.T) {
	tests := []struct {
		name        string
		start       string
		end         string
		expectedErr string
	}{
		{
			name:        "invalid start time format",
			start:       "25:00",
			end:         "5:00PM",
			expectedErr: "invalid start time format",
		},
		{
			name:        "invalid end time format",
			start:       "9:00AM",
			end:         "invalid",
			expectedErr: "invalid end time format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			store := storage.NewMemoryStorage()
			logger, _ := zap.NewDevelopment()
			h := New(store, logger)

			reqBody := Request{
				Name:    "Schedule",
				Team:    "team",
				Members: []string{"Alice"},
				Days:    []string{"Monday"},
				Start:   tt.start,
				End:     tt.end,
			}

			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err = h.CreateSchedule(c)

			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &errResp)
			require.NoError(t, err)
			assert.Contains(t, errResp.Error, tt.expectedErr)
		})
	}
}

func TestCreateSchedule_StartAfterEnd(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	reqBody := Request{
		Name:    "Schedule",
		Team:    "team",
		Members: []string{"Alice"},
		Days:    []string{"Monday"},
		Start:   "5:00PM",
		End:     "9:00AM",
	}

	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/schedule", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.CreateSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "start time must be before end time")
}

func TestGetSchedule_Success(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	// Create a schedule first
	schedule := storage.Schedule{
		Name:    "Weekday Coverage",
		Members: []string{"Alice", "Bob", "Charlie"},
		Days:    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:   parseTime(t, "9:00AM"),
		End:     parseTime(t, "5:00PM"),
	}
	err := store.AddSchedule("backend-team", schedule)
	require.NoError(t, err)

	// Query for oncall member on Monday at 10:00 AM
	queryTime := time.Date(2025, 4, 28, 10, 0, 0, 0, time.UTC) // Monday
	req := httptest.NewRequest(http.MethodGet, "/schedule?team=backend-team&time="+queryTime.Format(time.RFC3339), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.GetSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Alice", response["oncall"]) // First member
}

func TestGetSchedule_MissingParameters(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedErr string
	}{
		{
			name:        "missing team",
			url:         "/schedule?time=2025-04-28T10:00:00Z",
			expectedErr: "team query parameter is required",
		},
		{
			name:        "missing time",
			url:         "/schedule?team=backend-team",
			expectedErr: "time query parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			store := storage.NewMemoryStorage()
			logger, _ := zap.NewDevelopment()
			h := New(store, logger)

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := h.GetSchedule(c)

			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var errResp ErrorResponse
			err = json.Unmarshal(rec.Body.Bytes(), &errResp)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedErr, errResp.Error)
		})
	}
}

func TestGetSchedule_InvalidTimeFormat(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	req := httptest.NewRequest(http.MethodGet, "/schedule?team=backend-team&time=invalid-time", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.GetSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "invalid time format")
}

func TestGetSchedule_TeamNotFound(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	queryTime := time.Date(2025, 4, 28, 10, 0, 0, 0, time.UTC)
	req := httptest.NewRequest(http.MethodGet, "/schedule?team=non-existent&time="+queryTime.Format(time.RFC3339), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.GetSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Error, "no oncall member found")
}

func TestGetSchedule_NoMatchingSchedule(t *testing.T) {
	e := echo.New()
	store := storage.NewMemoryStorage()
	logger, _ := zap.NewDevelopment()
	h := New(store, logger)

	// Create a schedule for weekdays
	schedule := storage.Schedule{
		Name:    "Weekday Coverage",
		Members: []string{"Alice"},
		Days:    []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Start:   parseTime(t, "9:00AM"),
		End:     parseTime(t, "5:00PM"),
	}
	err := store.AddSchedule("backend-team", schedule)
	require.NoError(t, err)

	// Query for Saturday (no schedule)
	queryTime := time.Date(2025, 4, 26, 10, 0, 0, 0, time.UTC) // Saturday
	req := httptest.NewRequest(http.MethodGet, "/schedule?team=backend-team&time="+queryTime.Format(time.RFC3339), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.GetSchedule(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestParseWeekday(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Weekday
		wantErr  bool
	}{
		{"Monday", time.Monday, false},
		{"monday", time.Monday, false},
		{"MONDAY", time.Monday, false},
		{"Tuesday", time.Tuesday, false},
		{"Sunday", time.Sunday, false},
		{"Saturday", time.Saturday, false},
		{"InvalidDay", time.Sunday, true},
		{"", time.Sunday, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseWeekday(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// parseTime is a helper function to parse time strings in tests
func parseTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.Kitchen, timeStr)
	require.NoError(t, err)
	return parsed
}
