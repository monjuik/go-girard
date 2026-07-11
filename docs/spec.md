# Go Girard specification

## How am I doing it? In the most boring way possible!

- KISS: as simple as possible. Keep the core minimal, clean, testable, and maintainable.
- Use external dependencies only when they are truly necessary.
  - https://github.com/yuin/goldmark for Markdown parsing
- DRY.
- Lean to CQRS, DDD, and Clean Architecture principles in Go.
- Configuration lives in a file. The app reads it at startup and keeps it in memory.

One executable file.

## UI

Minimalist server-rendered UI with Pico CSS and Chart.js. We can load them via CDN.

## DB

Embedded SQLite. Singleton writer.

Add columns only when needed for search, store auxiliary data in JSON-fields.

## Main entities

- Person. Name, birthday, photo, position, company, contacts, note, audit.
- Company. Name, country, audit.
- Campaign. Playbook with instructions. Code, name, version, type, enrollment limiters, steps, instructions:
  - Types: finite, recurring
  - Recurrence: anchor = person.birthday, interval = (years = 1)
  - Enrollment limiter: by = person.company count = 1
  - Steps are ordered and describe the intention for the given stage
- Enrollment. Represents a person's participation in a campaign. Current state, next action, intention:
  - States: active, completed, stopped
  - Intention guides my next action with this person.

Notes are edited and stored as Markdown.
Audit stores the modification history for the current entity: created, updated etc.

## MCP

Strictly read-only.

### list_due_intentions

Provides the queue of overdue intentions and intentions due today.

### get_intention_context

Provides context of the action: intention, person and notes, company, campaign, current step, and instructions.


## Expected target file structure

```
go-girard/
├── cmd/
│   └── web/
│       └── main.go
│
├── app/
│   ├── app.go
│   ├── server.go
│   ├── routes.go
│   ├── config.go
│   ├── templates.go
│   ├── db.go
│   └── migrations.go
│
├── contacts/
│   ├── person.go
│   ├── company.go
│   ├── commands.go
│   ├── queries.go
│   ├── sqlite.go
│   └── contacts_test.go
│
├── campaigns/
│   ├── campaign.go
│   ├── enrollment.go
│   ├── config.go
│   ├── commands.go
│   ├── queries.go
│   ├── sqlite.go
│   └── campaigns_test.go
│
├── assets/
│   ├── static/
│   └── templates/
│
├── migrations/
│
├── go.mod
└── README.md
```
