-- Create users table
CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(255) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  phone VARCHAR(50),
  slack_user_id VARCHAR(100),
  created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    updated_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW ()
);

-- Create teams table
CREATE TABLE IF NOT EXISTS teams (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) UNIQUE NOT NULL,
  description TEXT,
  created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    updated_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW ()
);

-- Create team memberships table (many-to-many relationship)
CREATE TABLE IF NOT EXISTS team_members (
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
  role VARCHAR(50) DEFAULT 'member', -- member, lead, admin
  created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    PRIMARY KEY (team_id, user_id)
);

-- Create schedules table
CREATE TABLE IF NOT EXISTS schedules (
  id SERIAL PRIMARY KEY,
  team_id INTEGER REFERENCES teams (id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  start_time TIME NOT NULL,
  end_time TIME NOT NULL,
  timezone VARCHAR(100) DEFAULT 'UTC',
  created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    updated_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    UNIQUE (team_id, name)
);

-- Create schedule_days table (which days of week the schedule applies)
CREATE TABLE IF NOT EXISTS schedule_days (
  schedule_id INTEGER REFERENCES schedules (id) ON DELETE CASCADE,
  day_of_week INTEGER NOT NULL CHECK (
    day_of_week >= 0
    AND day_of_week <= 6
  ), -- 0=Sunday, 6=Saturday
  PRIMARY KEY (schedule_id, day_of_week)
);

-- Create schedule_members table (members in rotation for a schedule)
CREATE TABLE IF NOT EXISTS schedule_members (
  id SERIAL PRIMARY KEY,
  schedule_id INTEGER REFERENCES schedules (id) ON DELETE CASCADE,
  user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
  position INTEGER NOT NULL, -- Order in rotation
  created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    UNIQUE (schedule_id, user_id),
    UNIQUE (schedule_id, position)
);

-- Create rotations table (tracks current rotation state)
CREATE TABLE IF NOT EXISTS rotations (
  id SERIAL PRIMARY KEY,
  schedule_id INTEGER REFERENCES schedules (id) ON DELETE CASCADE UNIQUE,
  current_user_id INTEGER REFERENCES users (id) ON DELETE SET NULL,
  current_position INTEGER DEFAULT 0,
  last_rotation_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    next_rotation_at TIMESTAMP
  WITH
    TIME ZONE,
    created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    updated_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW ()
);

-- Create schedule_overrides table (temporary coverage changes)
CREATE TABLE IF NOT EXISTS schedule_overrides (
  id SERIAL PRIMARY KEY,
  schedule_id INTEGER REFERENCES schedules (id) ON DELETE CASCADE,
  original_user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
  override_user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
  start_time TIMESTAMP
  WITH
    TIME ZONE NOT NULL,
    end_time TIMESTAMP
  WITH
    TIME ZONE NOT NULL,
    reason TEXT,
    created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    CHECK (end_time > start_time)
);

-- Create incidents table
CREATE TABLE IF NOT EXISTS incidents (
  id SERIAL PRIMARY KEY,
  title VARCHAR(500) NOT NULL,
  description TEXT,
  severity VARCHAR(50) DEFAULT 'medium', -- low, medium, high, critical
  status VARCHAR(50) DEFAULT 'open', -- open, acknowledged, resolved, closed
  team_id INTEGER REFERENCES teams (id) ON DELETE SET NULL,
  assigned_to INTEGER REFERENCES users (id) ON DELETE SET NULL,
  acknowledged_at TIMESTAMP
  WITH
    TIME ZONE,
    resolved_at TIMESTAMP
  WITH
    TIME ZONE,
    created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW (),
    updated_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW ()
);

-- Create incident_timeline table (activity log for incidents)
CREATE TABLE IF NOT EXISTS incident_timeline (
  id SERIAL PRIMARY KEY,
  incident_id INTEGER REFERENCES incidents (id) ON DELETE CASCADE,
  user_id INTEGER REFERENCES users (id) ON DELETE SET NULL,
  action VARCHAR(100) NOT NULL, -- created, acknowledged, escalated, resolved, commented
  details JSONB,
  created_at TIMESTAMP
  WITH
    TIME ZONE DEFAULT NOW ()
);

-- Create indexes for better query performance
CREATE INDEX idx_schedules_team_id ON schedules (team_id);

CREATE INDEX idx_schedule_members_schedule_id ON schedule_members (schedule_id);

CREATE INDEX idx_schedule_members_user_id ON schedule_members (user_id);

CREATE INDEX idx_rotations_schedule_id ON rotations (schedule_id);

CREATE INDEX idx_schedule_overrides_schedule_id ON schedule_overrides (schedule_id);

CREATE INDEX idx_schedule_overrides_time_range ON schedule_overrides (start_time, end_time);

CREATE INDEX idx_incidents_team_id ON incidents (team_id);

CREATE INDEX idx_incidents_assigned_to ON incidents (assigned_to);

CREATE INDEX idx_incidents_status ON incidents (status);

CREATE INDEX idx_incident_timeline_incident_id ON incident_timeline (incident_id);

CREATE INDEX idx_team_members_team_id ON team_members (team_id);

CREATE INDEX idx_team_members_user_id ON team_members (user_id);
