package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/1995parham-learning/oncall-schedule/internal/storage"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Handler handles HTTP requests for the on-call schedule API.
type Handler struct {
	storage storage.Storage
	logger  *zap.Logger
}

// New creates a new handler instance.
func New(storage storage.Storage, logger *zap.Logger) *Handler {
	return &Handler{
		storage: storage,
		logger:  logger,
	}
}

// Request represents the schedule creation request.
type Request struct {
	Name    string   `json:"name"`
	Team    string   `json:"team"`
	Members []string `json:"members"`
	Days    []string `json:"days"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// CreateSchedule handles schedule creation requests.
func (h *Handler) CreateSchedule(c echo.Context) error {
	var req Request

	if err := c.Bind(&req); err != nil {
		h.logger.Error("failed to bind request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
	}

	// Validate request
	if err := h.validateRequest(&req); err != nil {
		h.logger.Warn("invalid request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	var schedule storage.Schedule
	schedule.Name = req.Name
	schedule.Members = req.Members

	// Parse days
	for _, d := range req.Days {
		day, err := parseWeekday(d)
		if err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid day: %s", d)})
		}
		schedule.Days = append(schedule.Days, day)
	}

	// Parse times
	start, err := time.Parse(time.Kitchen, req.Start)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid start time format, use '3:04PM' format"})
	}
	schedule.Start = start

	end, err := time.Parse(time.Kitchen, req.End)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid end time format, use '3:04PM' format"})
	}
	schedule.End = end

	// Validate time range
	if !start.Before(end) {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "start time must be before end time"})
	}

	if err := h.storage.AddSchedule(req.Team, schedule); err != nil {
		h.logger.Error("failed to add schedule", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create schedule"})
	}

	h.logger.Info("schedule created",
		zap.String("team", req.Team),
		zap.String("name", req.Name),
		zap.Strings("members", req.Members),
	)

	return c.NoContent(http.StatusCreated)
}

// GetSchedule handles schedule retrieval requests.
func (h *Handler) GetSchedule(c echo.Context) error {
	team := c.QueryParam("team")
	if team == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "team query parameter is required"})
	}

	timeStr := c.QueryParam("time")
	if timeStr == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "time query parameter is required"})
	}

	askTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid time format, use RFC3339 format"})
	}

	// Use the new GetCurrentOncall method which returns the currently oncall person
	oncall, found, err := h.storage.GetCurrentOncall(team, askTime)
	if err != nil {
		h.logger.Error("failed to get current oncall", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to retrieve oncall information"})
	}

	if !found {
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: "no oncall member found for the given time"})
	}

	h.logger.Info("oncall member found",
		zap.String("team", team),
		zap.String("oncall", oncall),
		zap.Time("time", askTime),
	)

	// Return single oncall member instead of array
	return c.JSON(http.StatusOK, map[string]string{"oncall": oncall})
}

// validateRequest validates the schedule creation request.
func (h *Handler) validateRequest(req *Request) error {
	if req.Team == "" {
		return fmt.Errorf("team is required")
	}

	if len(req.Members) == 0 {
		return fmt.Errorf("at least one member is required")
	}

	if len(req.Days) == 0 {
		return fmt.Errorf("at least one day is required")
	}

	if req.Start == "" {
		return fmt.Errorf("start time is required")
	}

	if req.End == "" {
		return fmt.Errorf("end time is required")
	}

	return nil
}

// parseWeekday parses a weekday string into time.Weekday.
func parseWeekday(day string) (time.Weekday, error) {
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		if strings.EqualFold(day, wd.String()) {
			return wd, nil
		}
	}
	return time.Sunday, fmt.Errorf("invalid weekday: %s", day)
}
