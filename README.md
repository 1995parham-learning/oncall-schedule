# Oncall Schedule

## Introduction

A production-ready Go application for managing on-call schedules and rotations. It provides a RESTful API for creating schedules, tracking rotations, and querying current on-call members.

## Features

### Storage Options
- **PostgreSQL Storage**: Production-ready persistence with full rotation tracking
- **In-Memory Storage**: Lightweight option for development and testing

### Core Capabilities
- Proper rotation tracking with position state management
- Support for multiple schedules per team
- Flexible weekday and time range configuration
- Timezone-aware time comparisons
- User and team management
- Database migrations for schema versioning
- Structured logging with Zap
- Graceful shutdown handling
- Configuration via YAML file or environment variables

## Configuration

The application can be configured using either a YAML file or environment variables.

### Configuration File

Create a `config.yaml` file (default included in the repository):

```yaml
server:
  address: "0.0.0.0"
  port: 1373

database:
  host: "localhost"
  port: 5432
  user: "oncall"
  password: "oncall"
  database: "oncall"
  ssl_mode: "disable"
  max_connections: 10
  min_connections: 2
  migrations_path: "migrations"
```

### Environment Variables

Configuration can be overridden using environment variables with the `ONCALL_` prefix:

```bash
# Server configuration
export ONCALL_SERVER__ADDRESS=localhost
export ONCALL_SERVER__PORT=8080

# Database configuration
export ONCALL_DATABASE__HOST=localhost
export ONCALL_DATABASE__PORT=5432
export ONCALL_DATABASE__USER=oncall
export ONCALL_DATABASE__PASSWORD=oncall
export ONCALL_DATABASE__DATABASE=oncall

# Storage mode (set to false to use in-memory storage)
export ONCALL_USE_DATABASE=true
```

Note: Use double underscores (`__`) to represent nested configuration keys.

### Default Values

**Server:**
- Address: `0.0.0.0`
- Port: `1373`

**Database:**
- Host: `localhost`
- Port: `5432`
- User: `oncall`
- Password: `oncall`
- Database: `oncall`
- SSL Mode: `disable`
- Max Connections: `10`
- Min Connections: `2`

## Quick Start

### Option 1: With PostgreSQL (Recommended for Production)

1. Start PostgreSQL using Docker Compose:

```bash
make db-up
```

Or manually:

```bash
docker-compose up -d postgres
```

2. Build and run the application:

```bash
make build
./bin/oncall-schedule
```

Or run directly:

```bash
make run
```

### Option 2: With In-Memory Storage (Development/Testing)

```bash
make run-memory
```

Or:

```bash
ONCALL_USE_DATABASE=false go run .
```

The API will be available at `http://localhost:1373` (or your configured address/port).

### Available Make Commands

```bash
make help          # Display all available commands
make build         # Build the application
make run           # Run with PostgreSQL
make run-memory    # Run with in-memory storage
make test          # Run tests with coverage
make db-up         # Start PostgreSQL
make db-down       # Stop PostgreSQL
make db-reset      # Reset database (deletes all data)
make lint          # Run linter
make clean         # Clean build artifacts
```

## API Endpoints

### 1. Create Schedule

Create a new on-call schedule for a team.

**Endpoint:** `POST /schedule`

**Request Body:**

```json
{
  "name": "Weekend Coverage",
  "team": "backend-team",
  "members": ["Alice", "Bob", "Charlie"],
  "days": ["Saturday", "Sunday"],
  "start": "9:00AM",
  "end": "5:00PM"
}
```

**Fields:**

- `name` (string, required): Schedule name/identifier
- `team` (string, required): Team identifier
- `members` (array, required): List of team members in the rotation (must not be empty)
- `days` (array, required): Weekdays when this schedule applies (case-insensitive: "Monday", "Tuesday", etc.)
- `start` (string, required): Start time in 12-hour format (e.g., "9:00AM", "1:30PM")
- `end` (string, required): End time in 12-hour format (must be after start time)

**Response:**

- `201 Created` on success
- `400 Bad Request` with error details on validation failure

**Example:**

```bash
curl -X POST http://localhost:1373/schedule \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Weekday Shift",
    "team": "ops-team",
    "members": ["John", "Jane", "Joe"],
    "days": ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"],
    "start": "9:00AM",
    "end": "5:00PM"
  }'
```

### 2. Get Current Oncall

Retrieve the currently on-call member for a team at a specific time.

**Endpoint:** `GET /schedule`

**Query Parameters:**

- `team` (string, required): Team identifier
- `time` (string, required): RFC3339 formatted timestamp (e.g., "2025-04-26T09:00:00Z")

**Response:**

- `200 OK` with current oncall member: `{"oncall": "John"}`
- `404 Not Found` if no schedule matches the query (wrong team, day, or time outside schedule window)
- `400 Bad Request` if parameters are missing or invalid

**Example:**

```bash
curl "http://localhost:1373/schedule?team=ops-team&time=2025-04-28T14:30:00Z"
```

**Response:**

```json
{
  "oncall": "John"
}
```

**Note:** With PostgreSQL storage, this returns the currently on-call person based on rotation state. With in-memory storage, it returns the first member in the rotation.

## How It Works

### Database Schema

The application uses a relational database schema with the following key tables:

- **users**: Stores user information (username, email, phone, Slack ID)
- **teams**: Team definitions
- **team_members**: Many-to-many relationship between teams and users
- **schedules**: Schedule definitions with time windows and team associations
- **schedule_days**: Which days of the week each schedule applies to
- **schedule_members**: Members in rotation for each schedule (with position tracking)
- **rotations**: Current rotation state for each schedule (tracks who's currently on-call)
- **schedule_overrides**: Temporary coverage changes (future feature)
- **incidents**: Incident tracking (future feature)
- **incident_timeline**: Activity log for incidents (future feature)

### Schedule Creation Flow

1. Validates all required fields are present and non-empty
2. Parses weekday strings (case-insensitive)
3. Parses start/end times in 12-hour format
4. Validates start time is before end time
5. Creates or retrieves team from database
6. Creates or retrieves users for each member
7. Creates schedule with time windows and days
8. Assigns members to schedule with rotation positions
9. Initializes rotation state (starts at position 0)

### Oncall Query Flow

1. Validates team and time parameters
2. Parses RFC3339 timestamp
3. Queries database for matching schedule:
   - Team matches
   - Day of week matches
   - Time falls within schedule window
4. Returns currently on-call member based on rotation state
5. Uses timezone-aware time comparisons

### Rotation Management

The rotation system tracks:
- **Current position**: Index into the members list
- **Current user**: Who is currently on-call
- **Last rotation time**: When the last rotation occurred
- **Next rotation time**: When the next rotation should happen (future feature)

The PostgreSQL storage implementation properly tracks rotation state, ensuring that the same person stays on-call until manually rotated. The in-memory storage always returns the first member (simplified implementation).

## Architecture

### Project Structure

```
oncall-schedule/
├── main.go                           # Application entry point with FX dependency injection
├── config.yaml                       # Default configuration
├── docker-compose.yml                # PostgreSQL setup for local development
├── Makefile                          # Build and development commands
├── migrations/                       # Database migration files
│   ├── 000001_initial_schema.up.sql
│   └── 000001_initial_schema.down.sql
└── internal/
    ├── config/                       # Configuration loading (YAML + env vars)
    │   └── config.go
    ├── db/                           # Database connection and migrations
    │   └── db.go
    ├── handler/                      # HTTP request handlers
    │   └── handler.go
    └── storage/                      # Storage interface and implementations
        ├── storage.go                # Interface and in-memory implementation
        └── postgres.go               # PostgreSQL implementation
```

### Technology Stack

- **Language**: Go 1.24+
- **Web Framework**: Echo v4
- **Database**: PostgreSQL with pgx driver
- **Migrations**: golang-migrate
- **Dependency Injection**: Uber FX
- **Logging**: Uber Zap
- **Configuration**: Koanf (YAML + environment variables)

### Design Patterns

- **Repository Pattern**: Storage interface with multiple implementations
- **Dependency Injection**: Clean separation of concerns using Uber FX
- **Clean Architecture**: Clear boundaries between layers (handler, storage, db)
- **Configuration Management**: 12-factor app approach with environment variables

## Roadmap to Complete Oncall Platform

This project is on a path to become a production-ready oncall platform. Here's what's implemented and what's planned:

### ✅ Phase 1: Foundation & Persistence (Completed)

- [x] PostgreSQL database layer with migrations
- [x] Proper rotation tracking with state management
- [x] User and team management (database schema)
- [x] Docker Compose for local development
- [x] Makefile for common tasks
- [x] Configuration management
- [x] Structured logging

### 🚧 Phase 2: Notifications & Alerting (Next)

- [ ] User contact information management API
- [ ] Email notification system (SMTP/SendGrid)
- [ ] Slack integration (webhooks + bot)
- [ ] SMS notifications (Twilio)
- [ ] Alert webhook endpoint
- [ ] Alert routing to current oncall person
- [ ] Notification delivery tracking

### 📋 Phase 3: Incident Management

- [ ] Incident creation and tracking
- [ ] Incident acknowledgment workflow
- [ ] Incident resolution workflow
- [ ] Escalation policies (auto-escalate if not acknowledged)
- [ ] Multi-level escalation chains
- [ ] Schedule override API (vacation/PTO handling)
- [ ] Shift swapping between team members

### 🔒 Phase 4: Production Readiness

- [ ] JWT-based authentication
- [ ] API key support for integrations
- [ ] Role-based access control (RBAC)
- [ ] Complete REST API (update/delete operations)
- [ ] API pagination and filtering
- [ ] OpenAPI/Swagger documentation
- [ ] Prometheus metrics
- [ ] Health check endpoints
- [ ] Rate limiting
- [ ] HTTPS/TLS configuration
- [ ] Dockerfile and Kubernetes manifests

### 📚 Phase 5: Testing & Documentation

- [ ] Unit tests for all packages
- [ ] Integration tests for database operations
- [ ] API endpoint tests
- [ ] Load testing
- [ ] Comprehensive API documentation
- [ ] Deployment guide
- [ ] Operations runbook

### 🎯 Future Enhancements

- Web UI for schedule management
- Mobile apps (iOS/Android)
- Calendar integrations (Google Calendar, Outlook)
- Advanced analytics and reporting
- Postmortem workflow
- Multi-region support
- SSO integration

## Contributing

This is a learning project, but contributions and suggestions are welcome! Feel free to:

- Open issues for bugs or feature requests
- Submit pull requests
- Share feedback on the architecture
- Suggest improvements to the roadmap

## License

Apache License 2.0 - See LICENSE file for details.
