# Go Girard specification

## How am I doing it? In the most boring way possible!

- KISS: as simple as possible. Keep the core minimal, clean, testable, and maintainable.
- Use external dependencies only when they are truly necessary. 
- DRY. 
- Lean to CQRS, DDD, and Clean Architecture principles in Go.
- Configuration lives in a file. The app reads it at startup and keeps it in memory.

One self-efficient excutable file.

## UI

Minimalistic server-rendered UI with Pico CSS and Chart.js.

## DB

Embedded SQLite. Singleton writer.

Add columns only when needed for search, store auxiliary data in JSON-fileds.

Complete an event and create the next one is a single DB transaction.

## Main entities

- Contact. Name, birthday, position, company, contacts, data, logs.
- Company. Name, country, data.
- Campaign. Code, name, version, type, enrollment limiters, steps, instructions.
  - Types: finite, recurring
  - Recurrence: anchor = contact.birthday, interval = (years = 1)
  - Enrollment limiter: by = contact.company count = 1
- Enrollment. Represents a contact's participation in a campaign.
  - States: active, completed, stopped
- Event. Represents either a scheduled action or a historical interaction.
  - States: planned, done, skipped
  - Outcome: positive, negative, call, meeting, not_relevant, not_now


## MCP

Strictly read-only. 

### list_due_events

Provides the queue: overdue events and events due today.

### get_event_context

Provides context of the event: event, contact, company, campaign and current step, instructions, communication history.
