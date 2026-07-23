package contacts

import (
	"context"
	"errors"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrPersonNotFound   = errors.New("person not found")
	ErrPersonNameExists = errors.New("person name already exists")

	ErrCompanyNotFound   = errors.New("company not found")
	ErrCompanyNameExists = errors.New("company name already exists")
)

// PersonCommands provides operations to change person data.
type PersonCommands interface {
	CreatePerson(ctx context.Context, input PersonInput) (common.ID, error)
	UpdatePerson(ctx context.Context, id common.ID, input PersonInput) error
}

// CompanyCommands provides operations to change company data.
type CompanyCommands interface {
	CreateCompany(ctx context.Context, input CompanyInput) (common.ID, error)
	UpdateCompany(ctx context.Context, id common.ID, input CompanyInput) error
	DeleteCompany(ctx context.Context, id common.ID) error
}
