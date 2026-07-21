package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/monjuik/go-girard/common"
	"github.com/monjuik/go-girard/contacts"
)

type recordingPersonQueries struct {
	filter   contacts.PersonsFilter
	rows     []contacts.PersonRowView
	id       common.ID
	person   contacts.PersonView
	err      error
	getCalls int
}

type recordingPersonCommands struct {
	createInput contacts.PersonInput
	createID    common.ID
	createErr   error
	createCalls int

	updateID    common.ID
	updateInput contacts.PersonInput
	updateErr   error
	updateCalls int
}

func (c *recordingPersonCommands) CreatePerson(
	ctx context.Context,
	input contacts.PersonInput,
) (common.ID, error) {
	c.createCalls++
	c.createInput = input
	return c.createID, c.createErr
}

func (c *recordingPersonCommands) UpdatePerson(
	ctx context.Context,
	id common.ID,
	input contacts.PersonInput,
) error {
	c.updateCalls++
	c.updateID = id
	c.updateInput = input
	return c.updateErr
}

func (q *recordingPersonQueries) ListPersonRows(
	ctx context.Context,
	filter contacts.PersonsFilter,
) ([]contacts.PersonRowView, error) {
	q.filter = filter
	return q.rows, nil
}

func (q *recordingPersonQueries) GetPerson(
	ctx context.Context,
	id common.ID,
) (contacts.PersonView, error) {
	q.getCalls++
	q.id = id
	return q.person, q.err
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

	server, err := NewServer(0, queries, &recordingPersonCommands{})
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

	if !strings.Contains(body, `href="/persons/new"`) {
		t.Fatal("GET /persons body does not contain new person URL")
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

	server, err := NewServer(0, queries, &recordingPersonCommands{})
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

	server, err := NewServer(0, queries, &recordingPersonCommands{})
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

func TestPersonPages(t *testing.T) {
	queries := &recordingPersonQueries{
		person: contacts.PersonView{
			ID:       "101",
			Name:     "Anna Petrova",
			Position: "Engineer",
		},
	}

	server, err := NewServer(0, queries, &recordingPersonCommands{})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/persons/new", nil)
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"GET /persons/new status = %d, want %d",
			response.Code,
			http.StatusOK,
		)
	}

	body := response.Body.String()
	for _, want := range []string{
		"New person",
		`action="/persons"`,
		"Create person",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /persons/new body does not contain %q", want)
		}
	}

	if count := strings.Count(body, "<!doctype html>"); count != 1 {
		t.Fatalf("GET /persons/new document count = %d, want 1", count)
	}

	request = httptest.NewRequest(
		http.MethodGet,
		"/persons/101?saved=1",
		nil,
	)
	response = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"GET /persons/101 status = %d, want %d",
			response.Code,
			http.StatusOK,
		)
	}
	if queries.id != common.ID(101) {
		t.Fatalf("GetPerson() id = %d, want 101", queries.id)
	}

	body = response.Body.String()
	for _, want := range []string{
		"Anna Petrova",
		"Engineer",
		`href="/persons/101/edit"`,
		"Person saved.",
		"Delete",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /persons/101 body does not contain %q", want)
		}
	}

	request = httptest.NewRequest(
		http.MethodGet,
		"/persons/101/edit",
		nil,
	)
	response = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf(
			"GET /persons/101/edit status = %d, want %d",
			response.Code,
			http.StatusOK,
		)
	}

	body = response.Body.String()
	for _, want := range []string{
		"Edit person",
		`action="/persons/101"`,
		`value="Anna Petrova"`,
		`value="Engineer"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf(
				"GET /persons/101/edit body does not contain %q",
				want,
			)
		}
	}

	queries.err = contacts.ErrPersonNotFound

	request = httptest.NewRequest(http.MethodGet, "/persons/999", nil)
	response = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"GET /persons/999 status = %d, want %d",
			response.Code,
			http.StatusNotFound,
		)
	}
}

func TestPersonPagesRejectInvalidID(t *testing.T) {
	queries := &recordingPersonQueries{}
	commands := &recordingPersonCommands{}

	server, err := NewServer(0, queries, commands)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	values := url.Values{"name": {"Anna Petrova"}}

	for _, id := range []string{"0", "-1", "invalid"} {
		t.Run(id, func(t *testing.T) {
			getCalls := queries.getCalls
			request := httptest.NewRequest(
				http.MethodGet,
				"/persons/"+id,
				nil,
			)
			response := httptest.NewRecorder()
			server.httpServer.Handler.ServeHTTP(response, request)

			if response.Code != http.StatusNotFound {
				t.Fatalf(
					"GET /persons/%s status = %d, want %d",
					id,
					response.Code,
					http.StatusNotFound,
				)
			}
			if queries.getCalls != getCalls {
				t.Fatalf("GET /persons/%s reached GetPerson", id)
			}

			updateCalls := commands.updateCalls
			response = postForm(
				server.httpServer.Handler,
				"/persons/"+id,
				values,
			)

			if response.Code != http.StatusNotFound {
				t.Fatalf(
					"POST /persons/%s status = %d, want %d",
					id,
					response.Code,
					http.StatusNotFound,
				)
			}
			if commands.updateCalls != updateCalls {
				t.Fatalf("POST /persons/%s reached UpdatePerson", id)
			}
		})
	}
}

func TestPersonFormRejectsLargeBody(t *testing.T) {
	commands := &recordingPersonCommands{}
	server, err := NewServer(
		0,
		&recordingPersonQueries{},
		commands,
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	tests := []struct {
		name  string
		path  string
		calls func() int
	}{
		{
			name: "create",
			path: "/persons",
			calls: func() int {
				return commands.createCalls
			},
		},
		{
			name: "update",
			path: "/persons/101",
			calls: func() int {
				return commands.updateCalls
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callsBefore := tt.calls()
			request := httptest.NewRequest(
				http.MethodPost,
				tt.path,
				strings.NewReader(
					"name="+strings.Repeat("a", maxPersonFormBodySize),
				),
			)
			request.Header.Set(
				"Content-Type",
				"application/x-www-form-urlencoded",
			)
			response := httptest.NewRecorder()

			server.httpServer.Handler.ServeHTTP(response, request)

			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf(
					"POST %s status = %d, want %d",
					tt.path,
					response.Code,
					http.StatusRequestEntityTooLarge,
				)
			}
			if tt.calls() != callsBefore {
				t.Fatalf("POST %s reached person command", tt.path)
			}
		})
	}
}

func TestCreatePerson(t *testing.T) {
	commands := &recordingPersonCommands{
		createID: common.ID(101),
	}

	server, err := NewServer(
		0,
		&recordingPersonQueries{},
		commands,
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	values := url.Values{
		"name":     {"  Anna Petrova  "},
		"position": {"  Engineer  "},
	}

	response := postForm(
		server.httpServer.Handler,
		"/persons",
		values,
	)

	if response.Code != http.StatusSeeOther {
		t.Fatalf(
			"POST /persons status = %d, want %d",
			response.Code,
			http.StatusSeeOther,
		)
	}
	if location := response.Header().Get("Location"); location != "/persons/101?saved=1" {
		t.Fatalf("POST /persons Location = %q", location)
	}

	wantInput := contacts.PersonInput{
		Name:     "  Anna Petrova  ",
		Position: "  Engineer  ",
	}
	if commands.createInput != wantInput {
		t.Fatalf(
			"CreatePerson() input = %+v, want %+v",
			commands.createInput,
			wantInput,
		)
	}

	for _, tt := range []struct {
		err     error
		message string
	}{
		{
			err:     contacts.ErrPersonNameRequired,
			message: "Name is required",
		},
		{
			err:     contacts.ErrPersonNameExists,
			message: "A person with this name already exists",
		},
	} {
		commands.createErr = tt.err

		response = postForm(
			server.httpServer.Handler,
			"/persons",
			values,
		)

		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf(
				"POST /persons status = %d, want %d",
				response.Code,
				http.StatusUnprocessableEntity,
			)
		}

		body := response.Body.String()
		if !strings.Contains(body, tt.message) {
			t.Fatalf("POST /persons body does not contain %q", tt.message)
		}
		if !strings.Contains(body, `value="  Anna Petrova  "`) {
			t.Fatal("POST /persons body does not preserve name")
		}
	}

	commands.createErr = nil
	callsBefore := commands.createCalls

	request := httptest.NewRequest(
		http.MethodPost,
		"/persons",
		strings.NewReader(values.Encode()),
	)
	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	request.Header.Set("Origin", "https://example.net")

	response = httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf(
			"cross-origin POST status = %d, want %d",
			response.Code,
			http.StatusForbidden,
		)
	}
	if commands.createCalls != callsBefore {
		t.Fatal("cross-origin POST reached CreatePerson")
	}
}

func postForm(
	handler http.Handler,
	path string,
	values url.Values,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(
		http.MethodPost,
		path,
		strings.NewReader(values.Encode()),
	)
	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestUpdatePerson(t *testing.T) {
	commands := &recordingPersonCommands{}

	server, err := NewServer(
		0,
		&recordingPersonQueries{},
		commands,
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	values := url.Values{
		"name":     {"  Anna Petrova  "},
		"position": {"  Director  "},
	}

	response := postForm(
		server.httpServer.Handler,
		"/persons/101",
		values,
	)

	if response.Code != http.StatusSeeOther {
		t.Fatalf(
			"POST /persons/101 status = %d, want %d",
			response.Code,
			http.StatusSeeOther,
		)
	}
	if location := response.Header().Get("Location"); location != "/persons/101?saved=1" {
		t.Fatalf("POST /persons/101 Location = %q", location)
	}

	if commands.updateID != common.ID(101) {
		t.Fatalf("UpdatePerson() id = %d, want 101", commands.updateID)
	}

	wantInput := contacts.PersonInput{
		Name:     "  Anna Petrova  ",
		Position: "  Director  ",
	}
	if commands.updateInput != wantInput {
		t.Fatalf(
			"UpdatePerson() input = %+v, want %+v",
			commands.updateInput,
			wantInput,
		)
	}

	commands.updateErr = contacts.ErrPersonNameExists

	response = postForm(
		server.httpServer.Handler,
		"/persons/101",
		values,
	)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf(
			"duplicate update status = %d, want %d",
			response.Code,
			http.StatusUnprocessableEntity,
		)
	}

	body := response.Body.String()
	for _, want := range []string{
		"A person with this name already exists",
		`action="/persons/101"`,
		`value="  Anna Petrova  "`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("duplicate update body does not contain %q", want)
		}
	}

	commands.updateErr = contacts.ErrPersonNotFound

	response = postForm(
		server.httpServer.Handler,
		"/persons/101",
		values,
	)

	if response.Code != http.StatusNotFound {
		t.Fatalf(
			"missing update status = %d, want %d",
			response.Code,
			http.StatusNotFound,
		)
	}
}

func FuzzPersonFormEndpoints(f *testing.F) {
	for _, seed := range []struct {
		endpoint uint8
		name     string
		position string
	}{
		{0, "Anna Petrova", "Engineer"},
		{1, "  Мария 李  ", "  Director  "},
		{0, "", ""},
		{1, "<script>alert(1)</script>", "%_&=+"},
		{
			0,
			strings.Repeat("a", maxPersonFormBodySize),
			"",
		},
	} {
		f.Add(seed.endpoint, seed.name, seed.position)
	}

	commands := &recordingPersonCommands{
		createID: common.ID(101),
	}
	server, err := NewServer(
		0,
		&recordingPersonQueries{},
		commands,
	)
	if err != nil {
		f.Fatalf("NewServer() error = %v", err)
	}

	f.Fuzz(func(t *testing.T, endpoint uint8, name, position string) {
		values := url.Values{
			"name":     {name},
			"position": {position},
		}
		encoded := values.Encode()

		path := "/persons"
		update := endpoint%2 == 1
		if update {
			path = "/persons/101"
		}

		createCallsBefore := commands.createCalls
		updateCallsBefore := commands.updateCalls

		request := httptest.NewRequest(
			http.MethodPost,
			path,
			strings.NewReader(encoded),
		)
		request.Header.Set(
			"Content-Type",
			"application/x-www-form-urlencoded",
		)

		response := httptest.NewRecorder()
		server.httpServer.Handler.ServeHTTP(response, request)

		if len(encoded) > maxPersonFormBodySize {
			if response.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf(
					"POST %s with %d-byte body status = %d, want %d",
					path,
					len(encoded),
					response.Code,
					http.StatusRequestEntityTooLarge,
				)
			}

			if commands.createCalls != createCallsBefore ||
				commands.updateCalls != updateCallsBefore {
				t.Fatal("oversized request reached person command")
			}
			return
		}

		if response.Code != http.StatusSeeOther {
			t.Fatalf(
				"POST %s status = %d, want %d",
				path,
				response.Code,
				http.StatusSeeOther,
			)
		}

		wantInput := contacts.PersonInput{
			Name:     name,
			Position: position,
		}

		if update {
			if commands.updateCalls != updateCallsBefore+1 {
				t.Fatal("request did not reach UpdatePerson exactly once")
			}
			if commands.createCalls != createCallsBefore {
				t.Fatal("update request reached CreatePerson")
			}
			if commands.updateID != common.ID(101) {
				t.Fatalf("UpdatePerson() ID = %d, want 101", commands.updateID)
			}
			if commands.updateInput != wantInput {
				t.Fatalf(
					"UpdatePerson() input = %+v, want %+v",
					commands.updateInput,
					wantInput,
				)
			}
			return
		}

		if commands.createCalls != createCallsBefore+1 {
			t.Fatal("request did not reach CreatePerson exactly once")
		}
		if commands.updateCalls != updateCallsBefore {
			t.Fatal("create request reached UpdatePerson")
		}
		if commands.createInput != wantInput {
			t.Fatalf(
				"CreatePerson() input = %+v, want %+v",
				commands.createInput,
				wantInput,
			)
		}
	})
}
