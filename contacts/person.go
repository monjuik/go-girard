package contacts

import (
	"errors"
	"strings"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrPersonIDInvalid    = errors.New("person id is invalid")
	ErrPersonNameRequired = errors.New("person name is required")
)

// Person represents a person domain entity.
type Person struct {
	id       common.ID
	name     string
	position string
	company  *Company
}

// PersonInput contains editable fields to add or update person data.
type PersonInput struct {
	Name     string
	Position string
}

// PersonsFilter controls searching and paging in the persons list.
type PersonsFilter struct {
	Query string
	Skip  int
	Limit int
}

// PersonRowView contains person data displayed in the table.
type PersonRowView struct {
	ID       string
	Name     string
	Position string
	Company  string
}

// PersonView contains data to show on an individual page.
type PersonView struct {
	ID       string
	Name     string
	Position string
}

func NewPerson(
	id common.ID,
	name string,
	position string,
	company *Company,
) (Person, error) {
	if !id.IsValid() {
		return Person{}, ErrPersonIDInvalid
	}

	person := Person{
		id:      id,
		company: company,
	}

	if err := person.Update(name, position); err != nil {
		return Person{}, err
	}

	return person, nil
}

func (p *Person) Update(name string, position string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrPersonNameRequired
	}

	p.name = name
	p.position = strings.TrimSpace(position)
	return nil
}

func (p Person) ID() common.ID {
	return p.id
}

func (p Person) Name() string {
	return p.name
}

func (p Person) Position() string {
	return p.position
}

func (p Person) Company() *Company {
	return p.company
}

type Company struct {
	id   common.ID
	name string
}

func NewCompany(id common.ID, name string) (Company, error) {
	if id.IsZero() {
		return Company{}, errors.New("id cannot be 0")
	}

	if strings.TrimSpace(name) == "" {
		return Company{}, errors.New("name cannot be empty")
	}

	return Company{
		id:   id,
		name: name,
	}, nil
}

func (c Company) ID() common.ID {
	return c.id
}

func (c Company) Name() string {
	return c.name
}
