package contacts

import (
	"context"

	"github.com/monjuik/go-girard/common"
)

type PersonsFilter struct {
}

type PersonQueries interface {
	ListPersonRows(ctx context.Context, filter PersonsFilter) ([]PersonRowView, error)
}

type StubPersonQueries struct{}

func NewStubPersonQueries() StubPersonQueries {
	return StubPersonQueries{}
}

func (q StubPersonQueries) ListPersonRows(ctx context.Context, filter PersonsFilter) ([]PersonRowView, error) {
	return []PersonRowView{
		{
			ID:       common.MustIDFromString("1992328621821009920").String(),
			Name:     "Anna Petrova",
			Position: "Head of Operations",
			Company:  "Northwind Logistics",
		},
		{
			ID:       common.MustIDFromString("1992328621821009921").String(),
			Name:     "Mark Jensen",
			Position: "Founder",
			Company:  "",
		},
	}, nil
}
