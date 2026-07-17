package app

import (
	"context"

	"github.com/monjuik/go-girard/contacts"
)

type App struct {
	personQueries contacts.PersonQueries
}

func NewApp(personQueries contacts.PersonQueries) *App {
	return &App{
		personQueries: personQueries,
	}
}

func (a *App) ListPersonRows(ctx context.Context, filter contacts.PersonsFilter) ([]contacts.PersonRowView, error) {
	return a.personQueries.ListPersonRows(ctx, filter)
}
