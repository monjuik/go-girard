package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monjuik/go-girard/contacts"
)

type recordingPersonQueries struct {
	filter contacts.PersonsFilter
	rows   []contacts.PersonRowView
}

func (q *recordingPersonQueries) ListPersonRows(
	ctx context.Context,
	filter contacts.PersonsFilter,
) ([]contacts.PersonRowView, error) {
	q.filter = filter
	return q.rows, nil
}

func TestPersonsPage(t *testing.T) {
	queries := &recordingPersonQueries{
		rows: []contacts.PersonRowView{
			{
				ID:       "101",
				Name:     "Anna Petrova",
				Position: "Head of Operations",
				Company:  "Northwind Logistics",
			},
		},
	}

	server, err := NewServer(0, queries)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/persons", nil)
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /persons status = %d, want %d", response.Code, http.StatusOK)
	}

	contentType := response.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("GET /persons Content-Type = %q, want text/html", contentType)
	}

	body := response.Body.String()

	if !strings.Contains(body, "Position") {
		t.Fatal("GET /persons body does not contain table header")
	}

	if !strings.Contains(body, "Anna Petrova") {
		t.Fatal("GET /persons body does not contain person")
	}

	if !strings.Contains(body, "Northwind Logistics") {
		t.Fatal("GET /persons body does not contain company")
	}
}

func TestPersonsPageSearchAndPaging(t *testing.T) {
	queries := &recordingPersonQueries{}

	for i := 1; i <= personsPerPage+1; i++ {
		queries.rows = append(queries.rows, contacts.PersonRowView{
			ID:       fmt.Sprintf("%d", i),
			Name:     fmt.Sprintf("Person %02d", i),
			Position: "Position",
		})
	}

	server, err := NewServer(0, queries)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/persons?q=anna&skip=20",
		nil,
	)
	response := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	if queries.filter.Query != "anna" {
		t.Fatalf("filter.Query = %q, want %q", queries.filter.Query, "anna")
	}
	if queries.filter.Skip != 20 {
		t.Fatalf("filter.Skip = %d, want %d", queries.filter.Skip, 20)
	}
	if queries.filter.Limit != personsPerPage+1 {
		t.Fatalf(
			"filter.Limit = %d, want %d",
			queries.filter.Limit,
			personsPerPage+1,
		)
	}

	body := response.Body.String()

	if !strings.Contains(body, `value="anna"`) {
		t.Fatal("response does not preserve search query")
	}

	if !strings.Contains(body, `href="/persons?q=anna"`) {
		t.Fatal("response does not contain previous page URL")
	}

	if !strings.Contains(
		body,
		`href="/persons?q=anna&amp;skip=40"`,
	) {
		t.Fatal("response does not contain next page URL")
	}

	if strings.Contains(body, "Person 21") {
		t.Fatal("response contains lookahead row")
	}
}

func TestPersonsPageRejectsInvalidSkip(t *testing.T) {
	queries := &recordingPersonQueries{}

	server, err := NewServer(0, queries)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	for _, skip := range []string{"invalid", "-1"} {
		t.Run(skip, func(t *testing.T) {
			request := httptest.NewRequest(
				http.MethodGet,
				"/persons?skip="+skip,
				nil,
			)
			response := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf(
					"status = %d, want %d",
					response.Code,
					http.StatusBadRequest,
				)
			}
		})
	}
}
