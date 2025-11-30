-- Drop tables in reverse order to handle foreign key constraints
DROP TABLE IF EXISTS incident_timeline;

DROP TABLE IF EXISTS incidents;

DROP TABLE IF EXISTS schedule_overrides;

DROP TABLE IF EXISTS rotations;

DROP TABLE IF EXISTS schedule_members;

DROP TABLE IF EXISTS schedule_days;

DROP TABLE IF EXISTS schedules;

DROP TABLE IF EXISTS team_members;

DROP TABLE IF EXISTS teams;

DROP TABLE IF EXISTS users;
