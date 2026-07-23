package contacts

import (
	"context"
	"fmt"

	"github.com/monjuik/go-girard/common"
)

type PersonService struct {
	persons PersonRepository
}

type CompanyService struct {
	companies CompanyRepository
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

func NewCompanyService(companies CompanyRepository) *CompanyService {
	return &CompanyService{companies: companies}
}

func (s *CompanyService) CreateCompany(ctx context.Context, input CompanyInput) (common.ID, error) {
	company, err := NewCompany(common.NewID(), input.Name, input.Country)
	if err != nil {
		return 0, err
	}
	if err := s.companies.Add(ctx, company); err != nil {
		return 0, fmt.Errorf("add company: %w", err)
	}
	return company.ID(), nil
}

func (s *CompanyService) UpdateCompany(
	ctx context.Context,
	id common.ID,
	input CompanyInput,
) error {
	company, err := NewCompany(
		id,
		input.Name,
		input.Country,
	)
	if err != nil {
		return err
	}

	if err := s.companies.Save(ctx, company); err != nil {
		return fmt.Errorf("save company: %w", err)
	}

	return nil
}

func (s *CompanyService) DeleteCompany(
	ctx context.Context,
	id common.ID,
) error {
	if !id.IsValid() {
		return ErrCompanyIDInvalid
	}

	if err := s.companies.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete company: %w", err)
	}

	return nil
}
