# Go Girard

Single-tenant self-hosted app for managing personal outreach campaigns and follow-up queue.

There are three main ideas:
- You keep notes for every person you are reaching out to: context, history, pains, facts, ideas, and anything else worth remembering, like Joe Girard's card system.
- A campaign provides guidance on the next steps for a given person.
- You see the follow-ups that are best to handle today.

## Why am I doing it?

I couldn't find a proper tool to manage sales-outreach activities. So I am building this tool for myself.

## What's in this name?

Joe Girard was an American salesman recognised by the Guinness Book of World Records as the seller of the most cars in a year (1,425 in 1973). This app is written in Go.

## What does this app do?

- Stores data about persons you are in touch. Name of a person should be unique. This is made on purpose. Better rename second John Smith in your app than confuse him by sending second welcome-message.

## Development

Run all tests:

  ```bash
  go test ./...
  ```

  ### Fuzz testing

  The project includes fuzz tests for ID parsing, person domain invariants,
  and HTTP form handling.

  Run a specific fuzz target for a limited time:

  ```bash
  go test ./common -fuzz=FuzzIDFromString -fuzztime=10s
  go test ./contacts -fuzz=FuzzPersonUpdate -fuzztime=10s
  go test ./app -fuzz=FuzzPersonFormEndpoints -fuzztime=10s
  ```
