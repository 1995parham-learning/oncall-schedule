package main

import (
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var storage map[string]Team

type Team struct {
	Schedules []Schedule
}

type Schedule struct {
	Name    string
	Members []string
	Days    []time.Weekday
	Start   time.Time
	End     time.Time
}

type Request struct {
	Name    string   `json:"name,omitempty"`
	Team    string   `json:"team"`
	Members []string `json:"members"`
	Days    []string `json:"days"`
	Start   string   `json:"start"`
	End     string   `json:"end"`
}

func createSchedule(c echo.Context) error {
	var req Request
	var schedule Schedule

	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}

	for _, d := range req.Days {
		if strings.EqualFold(d, time.Saturday.String()) {
			schedule.Days = append(schedule.Days, time.Saturday)
			continue
		}

		if strings.EqualFold(d, time.Sunday.String()) {
			schedule.Days = append(schedule.Days, time.Sunday)
			continue
		}

		// TODO

		return echo.ErrBadRequest
	}

	start, err := time.Parse(time.Kitchen, req.Start)
	if err != nil {
		return echo.ErrBadRequest
	}
	schedule.Start = start

	end, err := time.Parse(time.Kitchen, req.End)
	if err != nil {
		return echo.ErrBadRequest
	}
	schedule.End = end

	schedule.Name = req.Name
	schedule.Members = req.Members

	team := storage[req.Team]
	team.Schedules = append(team.Schedules, schedule)
	storage[req.Team] = team

	log.Println(storage)

	return c.NoContent(http.StatusCreated)
}

func getSchedule(c echo.Context) error {
	team := c.QueryParam("team")
	if team == "" {
		return echo.ErrBadRequest
	}

	t := c.QueryParam("time")
	askTime, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return echo.ErrBadRequest
	}

	sc, ok := storage[team]
	if !ok {
		return echo.ErrNotFound
	}

	for _, sc := range sc.Schedules {
		if !slices.Contains(sc.Days, askTime.Weekday()) {
			continue
		}

		year, month, day := askTime.Date()

		startTime := time.Date(year, month, day, sc.Start.Hour(), sc.Start.Minute(), 0, 0, time.UTC)
		endTime := time.Date(year, month, day, sc.End.Hour(), sc.End.Minute(), 0, 0, time.UTC)

		if !askTime.Before(endTime) || !askTime.After(startTime) {
			continue
		}

		return c.JSON(http.StatusOK, sc.Members)
	}

	return echo.ErrNotFound
}

func main() {
	storage = make(map[string]Team)
	app := echo.New()

	app.POST("/schedule", createSchedule)
	app.GET("/schedule", getSchedule)

	if err := app.Start("0.0.0.0:1373"); err != nil {
		log.Fatal(err)
	}
}
