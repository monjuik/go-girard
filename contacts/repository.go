package contacts

import (
	"context"
)

type PersonRepository interface {
	Add(ctx context.Context, person Person) error
	Save(ctx context.Context, person Person) error
}
