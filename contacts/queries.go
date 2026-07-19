package contacts

import (
	"context"

	"github.com/monjuik/go-girard/common"
)

// PersonQueries provides read-only access to person data.
type PersonQueries interface {
	ListPersonRows(ctx context.Context, filter PersonsFilter) ([]PersonRowView, error)
	GetPerson(ctx context.Context, id common.ID) (PersonView, error)
}
