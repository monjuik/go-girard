package contacts

import (
	"errors"
	"strings"

	"github.com/monjuik/go-girard/common"
)

type Person struct {
	id       common.ID
	name     string
	position string
	company  *Company
	email    string
	phone    string
	note     string
}

type PersonRowView struct {
	ID       string
	Name     string
	Position string
	Company  string
}

func NewPerson(id common.ID,
	name string,
	position string,
	company *Company,
	email string,
	phone string,
	note string) (Person, error) {
	if id.IsZero() {
		return Person{}, errors.New("id cannot be 0")
	}

	if strings.TrimSpace(name) == "" {
		return Person{}, errors.New("name cannot be empty")
	}

	return Person{
		id:       id,
		name:     name,
		position: position,
		company:  company,
		email:    email,
		phone:    phone,
		note:     note,
	}, nil
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

func (p Person) Email() string {
	return p.email
}

func (p Person) Phone() string {
	return p.phone
}

func (p Person) Note() string {
	return p.note
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
