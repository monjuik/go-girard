package contacts

import (
	"errors"
	"testing"

	"github.com/monjuik/go-girard/common"
)

func TestNewCompany(t *testing.T) {
	tests := []struct {
		name        string
		id          common.ID
		companyName string
		country     string
		wantName    string
		wantCountry string
		wantErr     error
	}{
		{
			name:        "valid company",
			id:          common.ID(1),
			companyName: "  Acme ltd.  ",
			country:     "  Cyprus  ",
			wantName:    "Acme ltd.",
			wantCountry: "Cyprus",
		},
		{
			name:        "company without country",
			id:          common.ID(2),
			companyName: "Northwind Logistics",
			wantName:    "Northwind Logistics",
		},
		{
			name:        "zero id",
			companyName: "Acme ltd.",
			wantErr:     ErrCompanyIDInvalid,
		},
		{
			name:        "negative id",
			id:          common.ID(-1),
			companyName: "Acme ltd.",
			wantErr:     ErrCompanyIDInvalid,
		},
		{
			name:        "blank name",
			id:          common.ID(2),
			companyName: "   ",
			country:     "Cyprus",
			wantErr:     ErrCompanyNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			company, err := NewCompany(tt.id, tt.companyName, tt.country)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"NewCompany() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewCompany() error = %v", err)
			}
			if company.ID() != tt.id {
				t.Fatalf("company.ID() = %v, want %v", company.ID(), tt.id)
			}
			if company.Name() != tt.wantName {
				t.Fatalf(
					"company.Name() = %q, want %q",
					company.Name(),
					tt.wantName,
				)
			}
			if company.Country() != tt.wantCountry {
				t.Fatalf(
					"company.Country() = %q, want %q",
					company.Country(),
					tt.wantCountry,
				)
			}
		})
	}
}

func TestCompanyUpdate(t *testing.T) {
	company, err := NewCompany(
		common.ID(1),
		"Acme ltd.",
		"Cyprus",
	)
	if err != nil {
		t.Fatalf("NewCompany() error = %v", err)
	}

	err = company.Update("  Northwind  ", "  Denmark  ")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if company.Name() != "Northwind" {
		t.Fatalf("company.Name() = %q, want %q", company.Name(), "Northwind")
	}
	if company.Country() != "Denmark" {
		t.Fatalf(
			"company.Country() = %q, want %q",
			company.Country(),
			"Denmark",
		)
	}

	err = company.Update(" \t ", "France")
	if !errors.Is(err, ErrCompanyNameRequired) {
		t.Fatalf(
			"Update() error = %v, want ErrCompanyNameRequired",
			err,
		)
	}

	if company.Name() != "Northwind" || company.Country() != "Denmark" {
		t.Fatal("failed Update() changed company")
	}
}
