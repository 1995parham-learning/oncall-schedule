# Oncall Schedule

## Introduction

A Go application for managing on-call schedules. It provides a RESTful API with two main endpoints:

- Create schedules for teams with specific time windows and weekdays
- Query the current on-call members for a team at a given time

## Features

- Thread-safe in-memory storage for schedules
- Support for multiple schedules per team
- Flexible weekday and time range configuration
- Timezone-aware time comparisons
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
```

### Environment Variables

Configuration can be overridden using environment variables with the `ONCALL_` prefix:

```bash
export ONCALL_SERVER__ADDRESS=localhost
export ONCALL_SERVER__PORT=8080
```

Note: Use double underscores (`__`) to represent nested configuration keys.

### Default Values

- Address: `0.0.0.0`
- Port: `1373`

## Usage

Start the server:

```bash
./oncall-schedule
```

The API will be available at `http://localhost:1373` (or your configured address/port).

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

### 2. Get Current Schedule

Retrieve the on-call members for a team at a specific time.

**Endpoint:** `GET /schedule`

**Query Parameters:**

- `team` (string, required): Team identifier
- `time` (string, required): RFC3339 formatted timestamp (e.g., "2025-04-26T09:00:00Z")

**Response:**

- `200 OK` with array of member names: `["Member1", "Member2"]`
- `404 Not Found` if no schedule matches the query (wrong team, day, or time outside schedule window)
- `400 Bad Request` if parameters are missing or invalid

**Example:**

```bash
curl "http://localhost:1373/schedule?team=ops-team&time=2025-04-28T14:30:00Z"
```

**Response:**

```json
["John", "Jane", "Joe"]
```

## How It Works

### Schedule Creation

1. Validates all required fields are present and non-empty
2. Parses weekday strings (case-insensitive)
3. Parses start/end times in 12-hour format
4. Validates start time is before end time
5. Stores schedule in thread-safe in-memory storage
6. Multiple schedules can exist per team

### Schedule Retrieval

1. Checks if the team exists
2. Iterates through the team's schedules
3. Matches the weekday of the requested time
4. Constructs timezone-aware start/end times for comparison
5. Returns members if the requested time falls within the schedule window
6. Uses the timezone from the requested time for accurate comparisons
