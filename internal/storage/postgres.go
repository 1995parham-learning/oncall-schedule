package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/1995parham-learning/oncall-schedule/internal/db"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// PostgresStorage implements Storage interface with PostgreSQL backend.
type PostgresStorage struct {
	db  *db.DB
	log *zap.Logger
}

// NewPostgresStorage creates a new PostgreSQL storage instance.
func NewPostgresStorage(database *db.DB, logger *zap.Logger) *PostgresStorage {
	return &PostgresStorage{
		db:  database,
		log: logger.Named("postgres-storage"),
	}
}

// AddSchedule adds a schedule to a team.
func (s *PostgresStorage) AddSchedule(teamName string, schedule Schedule) error {
	ctx := context.Background()

	// Start a transaction
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get or create team
	var teamID int
	err = tx.QueryRow(ctx,
		`INSERT INTO teams (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`,
		teamName,
	).Scan(&teamID)
	if err != nil {
		return fmt.Errorf("failed to get/create team: %w", err)
	}

	// Get or create users for each member
	userIDs := make(map[string]int)
	for _, member := range schedule.Members {
		var userID int
		// For now, we'll use member name as both username and email
		// In a real system, these would be proper user objects
		err = tx.QueryRow(ctx,
			`INSERT INTO users (username, email) VALUES ($1, $2)
			 ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
			 RETURNING id`,
			member,
			fmt.Sprintf("%s@example.com", member),
		).Scan(&userID)
		if err != nil {
			return fmt.Errorf("failed to get/create user %s: %w", member, err)
		}
		userIDs[member] = userID

		// Add user to team if not already a member
		_, err = tx.Exec(ctx,
			`INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, $3)
			 ON CONFLICT (team_id, user_id) DO NOTHING`,
			teamID, userID, "member",
		)
		if err != nil {
			return fmt.Errorf("failed to add user to team: %w", err)
		}
	}

	// Insert schedule
	var scheduleID int
	err = tx.QueryRow(ctx,
		`INSERT INTO schedules (team_id, name, start_time, end_time, timezone)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		teamID,
		schedule.Name,
		schedule.Start.Format("15:04:05"),
		schedule.End.Format("15:04:05"),
		"UTC",
	).Scan(&scheduleID)
	if err != nil {
		return fmt.Errorf("failed to insert schedule: %w", err)
	}

	// Insert schedule days
	for _, day := range schedule.Days {
		_, err = tx.Exec(ctx,
			`INSERT INTO schedule_days (schedule_id, day_of_week) VALUES ($1, $2)`,
			scheduleID, int(day),
		)
		if err != nil {
			return fmt.Errorf("failed to insert schedule day: %w", err)
		}
	}

	// Insert schedule members with their position in rotation
	for position, member := range schedule.Members {
		userID := userIDs[member]
		_, err = tx.Exec(ctx,
			`INSERT INTO schedule_members (schedule_id, user_id, position)
			 VALUES ($1, $2, $3)`,
			scheduleID, userID, position,
		)
		if err != nil {
			return fmt.Errorf("failed to insert schedule member: %w", err)
		}
	}

	// Initialize rotation state for the schedule
	if len(schedule.Members) > 0 {
		firstUserID := userIDs[schedule.Members[0]]
		_, err = tx.Exec(ctx,
			`INSERT INTO rotations (schedule_id, current_user_id, current_position, last_rotation_at)
			 VALUES ($1, $2, $3, $4)`,
			scheduleID, firstUserID, 0, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to initialize rotation: %w", err)
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("schedule added successfully",
		zap.String("team", teamName),
		zap.String("schedule", schedule.Name),
		zap.Int("schedule_id", scheduleID),
	)

	return nil
}

// GetTeam retrieves a team's schedules.
func (s *PostgresStorage) GetTeam(teamName string) (Team, bool, error) {
	ctx := context.Background()

	// Get team ID
	var teamID int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id FROM teams WHERE name = $1`,
		teamName,
	).Scan(&teamID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Team{}, false, nil
		}
		return Team{}, false, fmt.Errorf("failed to get team: %w", err)
	}

	// Get all schedules for the team
	rows, err := s.db.Pool.Query(ctx,
		`SELECT id, name, start_time, end_time FROM schedules WHERE team_id = $1`,
		teamID,
	)
	if err != nil {
		return Team{}, false, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var scheduleID int
		var name string
		var startTime, endTime time.Time

		err = rows.Scan(&scheduleID, &name, &startTime, &endTime)
		if err != nil {
			return Team{}, false, fmt.Errorf("failed to scan schedule: %w", err)
		}

		// Get days for this schedule
		dayRows, err := s.db.Pool.Query(ctx,
			`SELECT day_of_week FROM schedule_days WHERE schedule_id = $1 ORDER BY day_of_week`,
			scheduleID,
		)
		if err != nil {
			return Team{}, false, fmt.Errorf("failed to query schedule days: %w", err)
		}

		var days []time.Weekday
		for dayRows.Next() {
			var day int
			if err = dayRows.Scan(&day); err != nil {
				dayRows.Close()
				return Team{}, false, fmt.Errorf("failed to scan day: %w", err)
			}
			days = append(days, time.Weekday(day))
		}
		dayRows.Close()

		// Get members for this schedule (in rotation order)
		memberRows, err := s.db.Pool.Query(ctx,
			`SELECT u.username
			 FROM schedule_members sm
			 JOIN users u ON sm.user_id = u.id
			 WHERE sm.schedule_id = $1
			 ORDER BY sm.position`,
			scheduleID,
		)
		if err != nil {
			return Team{}, false, fmt.Errorf("failed to query schedule members: %w", err)
		}

		var members []string
		for memberRows.Next() {
			var username string
			if err = memberRows.Scan(&username); err != nil {
				memberRows.Close()
				return Team{}, false, fmt.Errorf("failed to scan member: %w", err)
			}
			members = append(members, username)
		}
		memberRows.Close()

		schedules = append(schedules, Schedule{
			Name:    name,
			Members: members,
			Days:    days,
			Start:   startTime,
			End:     endTime,
		})
	}

	if err = rows.Err(); err != nil {
		return Team{}, false, fmt.Errorf("error iterating schedules: %w", err)
	}

	return Team{Schedules: schedules}, true, nil
}

// GetCurrentOncall returns the currently oncall member for a team at the specified time.
// This implements proper rotation logic instead of returning all members.
func (s *PostgresStorage) GetCurrentOncall(teamName string, at time.Time) (string, bool, error) {
	ctx := context.Background()

	// Get team ID
	var teamID int
	err := s.db.Pool.QueryRow(ctx,
		`SELECT id FROM teams WHERE name = $1`,
		teamName,
	).Scan(&teamID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get team: %w", err)
	}

	// Find matching schedule for the given time
	dayOfWeek := int(at.Weekday())
	timeOfDay := at.Format("15:04:05")

	var currentUserID *int
	var username string
	err = s.db.Pool.QueryRow(ctx,
		`SELECT r.current_user_id, u.username
		 FROM schedules s
		 JOIN schedule_days sd ON s.id = sd.schedule_id
		 JOIN rotations r ON s.id = r.schedule_id
		 LEFT JOIN users u ON r.current_user_id = u.id
		 WHERE s.team_id = $1
		   AND sd.day_of_week = $2
		   AND s.start_time <= $3::time
		   AND s.end_time >= $3::time
		 LIMIT 1`,
		teamID, dayOfWeek, timeOfDay,
	).Scan(&currentUserID, &username)

	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, fmt.Errorf("failed to get current oncall: %w", err)
	}

	if currentUserID == nil {
		return "", false, nil
	}

	return username, true, nil
}
