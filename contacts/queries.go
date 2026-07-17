package contacts

import (
	"context"
)

type PersonsFilter struct {
	Query string
	Skip  int
	Limit int
}

type PersonQueries interface {
	ListPersonRows(ctx context.Context, filter PersonsFilter) ([]PersonRowView, error)
}
