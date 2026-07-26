package contacts

import (
	"errors"
	"strings"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrPersonIDInvalid        = errors.New("person id is invalid")
	ErrPersonNameRequired     = errors.New("person name is required")
	ErrPersonCompanyIDInvalid = errors.New("person company id is invalid")
)

// Person represents a person domain entity.
type Person struct {
	id        common.ID
	name      string
	position  string
	companyID common.ID
	note      string
}

// PersonInput contains editable fields to add or update person data.
type PersonInput struct {
	Name      string
	Position  string
	CompanyID common.ID
	Note      string
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
	ID          string
	Name        string
	Position    string
	CompanyID   string
	CompanyName string
	Note        string
}

func NewPerson(
	id common.ID,
	name string,
	position string,
	companyID common.ID,
	note string,
) (Person, error) {
	if !id.IsValid() {
		return Person{}, ErrPersonIDInvalid
	}

	if !companyID.IsZero() && !companyID.IsValid() {
		return Person{}, ErrPersonCompanyIDInvalid
	}

	person := Person{
		id:        id,
		companyID: companyID,
	}

	if err := person.Update(name, position, note); err != nil {
		return Person{}, err
	}

	return person, nil
}

func (p *Person) Update(name, position, note string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrPersonNameRequired
	}

	p.name = name
	p.position = strings.TrimSpace(position)
	p.note = note
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

func (p Person) CompanyID() common.ID {
	return p.companyID
}

func (p Person) Note() string {
	return p.note
}
