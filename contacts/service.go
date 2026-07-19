package contacts

import (
	"context"
	"fmt"

	"github.com/monjuik/go-girard/common"
)

type PersonService struct {
	persons PersonRepository
}

func NewPersonService(persons PersonRepository) *PersonService {
	return &PersonService{
		persons: persons,
	}
}

func (s *PersonService) CreatePerson(
	ctx context.Context,
	input PersonInput,
) (common.ID, error) {
	person, err := NewPerson(
		common.NewID(),
		input.Name,
		input.Position,
		nil,
	)
	if err != nil {
		return 0, err
	}

	if err := s.persons.Add(ctx, person); err != nil {
		return 0, fmt.Errorf("add person: %w", err)
	}

	return person.ID(), nil
}

func (s *PersonService) UpdatePerson(
	ctx context.Context,
	id common.ID,
	input PersonInput,
) error {
	person, err := NewPerson(
		id,
		input.Name,
		input.Position,
		nil,
	)
	if err != nil {
		return err
	}

	if err := s.persons.Save(ctx, person); err != nil {
		return fmt.Errorf("save person: %w", err)
	}

	return nil
}
