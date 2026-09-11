package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/monjuik/go-girard/campaigns"
	"github.com/monjuik/go-girard/common"
	"github.com/monjuik/go-girard/contacts"
)

type mcpQueries struct {
	enrollments campaigns.EnrollmentQueries
	persons     contacts.PersonQueries
	companies   contacts.CompanyQueries
	campaigns   map[string]campaigns.Campaign
	today       func() common.Date
}

type personContextInput struct {
	PersonID string `json:"person_id" jsonschema:"String ID from person.id in list_due_intentions."`
}

func (s *Server) newMCPHandler(version string) http.Handler {
	server := newMCPServer(version, &mcpQueries{
		enrollments: s.enrollmentQueries,
		persons:     s.personQueries,
		companies:   s.companyQueries,
		campaigns:   s.campaigns,
		today:       func() common.Date { return s.today() },
	})
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
}

func newMCPServer(version string, queries *mcpQueries) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "go-girard",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_due_intentions",
		Description: `List active intentions overdue or due today in the server's local time zone.
Use person.id to call get_person_context for details.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: new(bool),
		},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, dueIntentions, error) {
		result, err := queries.listDueIntentions(ctx)
		return nil, result, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_person_context",
		Description: `Get a person's details, Markdown note, company, and all enrollments,
including their state, intention, campaign, and current step instructions.
Includes future and inactive enrollments.
Missing campaign or step configuration leaves only the available data and stored codes.`,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: new(bool),
		},
	}, func(
		ctx context.Context,
		_ *mcp.CallToolRequest,
		input personContextInput,
	) (*mcp.CallToolResult, personContext, error) {
		result, err := queries.getPersonContext(ctx, input.PersonID)
		return nil, result, err
	})

	return server
}

type personContext struct {
	Person      mcpPerson          `json:"person"`
	Company     *mcpCompany        `json:"company,omitempty"`
	Enrollments []personEnrollment `json:"enrollments"`
}

type personEnrollment struct {
	EnrollmentID string                    `json:"enrollment_id"`
	State        campaigns.EnrollmentState `json:"state"`
	Next         string                    `json:"next,omitempty"`
	Intention    string                    `json:"intention"`
	Campaign     mcpCampaign               `json:"campaign"`
	Step         mcpStep                   `json:"step"`
}

type mcpPerson struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Note     string `json:"note"`
}

type mcpCompany struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

type mcpCampaign struct {
	Code        string `json:"code"`
	Name        string `json:"name,omitempty"`
	Version     int    `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
}

type mcpStep struct {
	Code         string `json:"code"`
	Name         string `json:"name,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

type dueIntentions struct {
	AsOf       string         `json:"as_of"`
	Intentions []dueIntention `json:"intentions"`
}

type dueIntention struct {
	EnrollmentID string       `json:"enrollment_id"`
	Next         string       `json:"next"`
	Status       string       `json:"status"`
	Intention    string       `json:"intention"`
	Person       mcpPersonRef `json:"person"`
	Campaign     mcpConfigRef `json:"campaign"`
	Step         mcpConfigRef `json:"step"`
}

type mcpPersonRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// mcpConfigRef identifies a campaign or step.
// Name is absent when its configuration is missing.
type mcpConfigRef struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

func (q *mcpQueries) getPersonContext(
	ctx context.Context,
	personID string,
) (personContext, error) {
	id, err := common.IDFromString(personID)
	if err != nil {
		return personContext{}, fmt.Errorf("invalid person_id: %w", err)
	}

	person, err := q.persons.GetPerson(ctx, id)
	if err != nil {
		return personContext{}, fmt.Errorf("get person: %w", err)
	}

	rows, err := q.enrollments.ListPersonEnrollments(ctx, id)
	if err != nil {
		return personContext{}, fmt.Errorf("list person enrollments: %w", err)
	}

	result := personContext{
		Person: mcpPerson{
			ID:       person.ID,
			Name:     person.Name,
			Position: person.Position,
			Note:     person.Note,
		},
		Enrollments: make([]personEnrollment, 0, len(rows)),
	}

	for _, row := range rows {
		enrollment := personEnrollment{
			EnrollmentID: row.ID,
			State:        row.State,
			Next:         row.Next.String(),
			Intention:    row.Intention,
			Campaign:     mcpCampaign{Code: row.Campaign},
			Step:         mcpStep{Code: row.Step},
		}

		if campaign, exists := q.campaigns[row.Campaign]; exists {
			enrollment.Campaign.Name = campaign.Name()
			enrollment.Campaign.Version = campaign.Version()
			enrollment.Campaign.Description = campaign.Description()

			if step, exists := campaign.FindStep(row.Step); exists {
				enrollment.Step.Name = step.Name()
				enrollment.Step.Instructions = step.Instructions()
			}
		}

		result.Enrollments = append(result.Enrollments, enrollment)
	}

	if person.CompanyID != "" {
		companyID, err := common.IDFromString(person.CompanyID)
		if err != nil {
			return personContext{}, fmt.Errorf("parse company id: %w", err)
		}

		company, err := q.companies.GetCompany(ctx, companyID)
		if err != nil {
			return personContext{}, fmt.Errorf("get company: %w", err)
		}

		result.Company = &mcpCompany{
			ID:      company.ID,
			Name:    company.Name,
			Country: company.Country,
		}
	}

	return result, nil
}

func (q *mcpQueries) listDueIntentions(
	ctx context.Context,
) (dueIntentions, error) {
	today := q.today()

	rows, err := q.enrollments.ListDueEnrollments(ctx, today)
	if err != nil {
		return dueIntentions{}, fmt.Errorf("list due enrollments: %w", err)
	}

	result := dueIntentions{
		AsOf:       today.String(),
		Intentions: make([]dueIntention, 0, len(rows)),
	}

	for _, row := range rows {
		item := dueIntention{
			EnrollmentID: row.ID,
			Next:         row.Next.String(),
			Status:       "due_today",
			Intention:    row.Intention,
			Person: mcpPersonRef{
				ID:   row.PersonID,
				Name: row.PersonName,
			},
			Campaign: mcpConfigRef{Code: row.Campaign},
			Step:     mcpConfigRef{Code: row.Step},
		}

		if row.Next.IsBefore(today) {
			item.Status = "overdue"
		}

		if campaign, exists := q.campaigns[row.Campaign]; exists {
			item.Campaign.Name = campaign.Name()

			if step, exists := campaign.FindStep(row.Step); exists {
				item.Step.Name = step.Name()
			}
		}

		result.Intentions = append(result.Intentions, item)
	}

	return result, nil
}
