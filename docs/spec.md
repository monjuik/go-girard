# Go Girard specification

## How am I doing it? In the most boring way possible!

- KISS: as simple as possible. Keep the core minimal, clean, testable, and maintainable
- Use external dependencies only when they are truly necessary
  - https://github.com/yuin/goldmark for Markdown parsing
- DRY
- Lean to CQRS, DDD, and Clean Architecture principles in Go
- Product configuration lives in a file. The app reads it at startup and keeps it in memory
- Technical configuration uses flags with default values

One executable file.

## UI

Minimalist server-rendered UI with Pico CSS and Chart.js. We can load them via CDN.

## DB

Embedded SQLite. Singleton writer.

Add columns only when needed for search, store auxiliary data in JSON-fields.

### IDs

Entity IDs use Snowflake-style 64-bit integers, reasons:
- SQLite stores them efficiently as `INTEGER`
- IDs are generated in the application, keeps them unpredictable for MCP
- Snowflake IDs are time-sortable, which is useful for default ordering and indexes
- UUIDv7 is 128-bit and would require `TEXT` or `BLOB(16)` storage in SQLite

Useful commands to work with the db:
```bash
sqlite3 'file:go-girard.db?mode=ro'
sqlite3 'file:go-girard.db'
sqlite> .headers on
sqlite> .mode box
sqlite> .tables
sqlite> SELECT * FROM migration;
```

Test data:
```sql
INSERT INTO company (id, name)
VALUES (1, 'Northwind Logistics');

INSERT INTO person (id, name, position, company)
VALUES (101, 'Anna Petrova', 'Head of Operations', 1);
```

### DB backup

Create a backup:
`sqlite3 go-girard.db ".backup 'go-girard-backup-$(date +%Y%m%d-%H%M%S).db'"`

Check file:
`sqlite3 go-girard-backup-*.db "PRAGMA integrity_check;"`

## Main entities

- Person. Name, birthday, photo, position, company, contacts, note, audit
- Company. Name, country, audit
- Campaign. Playbook with instructions. Code, name, version, type, steps, instructions:
  - Types: finite, recurring
  - Recurrence: anchor = person.birthday, interval = (years = 1)
  - Steps are ordered and describe the intention for the given stage
- Enrollment. Represents a person's participation in a campaign. Current state, next action, intention:
  - States: active, completed, stopped
  - Intention guides my next action with this person

Notes are edited and stored as Markdown.
Audit stores the modification history for the current entity: created, updated etc.

## MCP

Strictly read-only.

Available at `/mcp` on the web server's port using stateless Streamable HTTP
with JSON responses. Starts with the UI; no separate authentication.

### list_due_intentions

Provides the queue of overdue intentions and intentions due today.

### get_person_context

Accepts `person_id` from a `list_due_intentions` item's `person.id`.
Provides the person's details. Access is not restricted to the due queue.

## Expected target file structure

```
go-girard/
├── cmd/
│   └── web/
│       └── main.go
│
├── app/
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


## TODO

- [ ] company page: markdown note
- [ ] company page: a list of persons
- [ ] graceful shutdown of the db files after getting sigkill


## Releases

GitHub Releases provide archives for Linux and macOS on amd64 and arm64. Each
archive contains the executable, this README, the license, and an example
configuration. Use the published `SHA256SUMS` file to verify downloads.

To publish a semantic version after the release workflow is present on `main`,
create and push an annotated tag:

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

The tag triggers tests, native CGO builds, and publication of the GitHub
Release with automatically generated release notes.
