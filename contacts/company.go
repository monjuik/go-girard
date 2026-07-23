package contacts

import (
	"errors"
	"strings"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrCompanyIDInvalid    = errors.New("company id is invalid")
	ErrCompanyNameRequired = errors.New("company name is required")
)

// Company represents a company domain entity.
type Company struct {
	id      common.ID
	name    string
	country string
}

// CompanyInput contains editable company fields.
type CompanyInput struct {
	Name    string
	Country string
}

// CompaniesFilter controls searching and paging in the companies list.
type CompaniesFilter struct {
	Query string
	Skip  int
	Limit int
}

// CompanyRowView contains company data displayed in the table.
type CompanyRowView struct {
	ID      string
	Name    string
	Country string
}

// CompanyView contains data displayed on an individual company page.
type CompanyView struct {
	ID      string
	Name    string
	Country string
}

func NewCompany(id common.ID, name string, country string) (Company, error) {
	if !id.IsValid() {
		return Company{}, ErrCompanyIDInvalid
	}
	company := Company{id: id}

	if err := company.Update(name, country); err != nil {
		return Company{}, err
	}

	return company, nil
}

func (c *Company) Update(name string, country string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrCompanyNameRequired
	}

	c.name = name
	c.country = strings.TrimSpace(country)
	return nil
}

func (c Company) ID() common.ID {
	return c.id
}

func (c Company) Name() string {
	return c.name
}

func (c Company) Country() string {
	return c.country
}
