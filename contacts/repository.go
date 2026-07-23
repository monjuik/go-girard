package contacts

import (
	"context"

	"github.com/monjuik/go-girard/common"
)

type PersonRepository interface {
	Add(ctx context.Context, person Person) error
	Save(ctx context.Context, person Person) error
}

type CompanyRepository interface {
	Add(ctx context.Context, company Company) error
	Save(ctx context.Context, company Company) error
	Delete(ctx context.Context, id common.ID) error
}
