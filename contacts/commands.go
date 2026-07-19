package contacts

import (
	"context"
	"errors"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrPersonNotFound   = errors.New("person not found")
	ErrPersonNameExists = errors.New("person name already exists")
)

// PersonCommands provides operations to change person data.
type PersonCommands interface {
	CreatePerson(ctx context.Context, input PersonInput) (common.ID, error)
	UpdatePerson(ctx context.Context, id common.ID, input PersonInput) error
}
