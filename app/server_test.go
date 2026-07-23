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

type recordingCompanyQueries struct {
	filter   contacts.CompaniesFilter
	rows     []contacts.CompanyRowView
	id       common.ID
	company  contacts.CompanyView
	err      error
	getCalls int
}

type recordingCompanyCommands struct {
	createInput contacts.CompanyInput
	createID    common.ID
	createErr   error
	createCalls int

	updateID    common.ID
	updateInput contacts.CompanyInput
	updateErr   error
	updateCalls int

	deleteID    common.ID
	deleteErr   error
	deleteCalls int
}

type serverFixture struct {
	handler         http.Handler
	personQueries   *recordingPersonQueries
	personCommands  *recordingPersonCommands
	companyQueries  *recordingCompanyQueries
	companyCommands *recordingCompanyCommands
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

func (q *recordingCompanyQueries) ListCompanyRows(
	ctx context.Context,
	filter contacts.CompaniesFilter,
) ([]contacts.CompanyRowView, error) {
	q.filter = filter
	return q.rows, q.err
}

func (q *recordingCompanyQueries) GetCompany(
	ctx context.Context,
	id common.ID,
) (contacts.CompanyView, error) {
	q.getCalls++
	q.id = id
	return q.company, q.err
}

func (c *recordingCompanyCommands) CreateCompany(
	ctx context.Context,
	input contacts.CompanyInput,
) (common.ID, error) {
	c.createCalls++
	c.createInput = input
	return c.createID, c.createErr
}

func (c *recordingCompanyCommands) UpdateCompany(
	ctx context.Context,
	id common.ID,
	input contacts.CompanyInput,
) error {
	c.updateCalls++
	c.updateID = id
	c.updateInput = input
	return c.updateErr
}

func (c *recordingCompanyCommands) DeleteCompany(
	ctx context.Context,
	id common.ID,
) error {
	c.deleteCalls++
	c.deleteID = id
	return c.deleteErr
}

func newServerFixture(t *testing.T) *serverFixture {
	t.Helper()

	fixture := &serverFixture{
		personQueries:   &recordingPersonQueries{},
		personCommands:  &recordingPersonCommands{},
		companyQueries:  &recordingCompanyQueries{},
		companyCommands: &recordingCompanyCommands{},
	}

	server, err := NewServer(
		0,
		fixture.personQueries,
		fixture.personCommands,
		fixture.companyQueries,
		fixture.companyCommands,
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	fixture.handler = server.httpServer.Handler
	return fixture
}

func (f *serverFixture) get(path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	return f.serve(request)
}

func (f *serverFixture) postForm(
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
	return f.serve(request)
}

func (f *serverFixture) serve(
	request *http.Request,
) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	return response
}

func assertStatus(
	t *testing.T,
	response *httptest.ResponseRecorder,
	want int,
) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("status = %d, want %d", response.Code, want)
	}
}

func assertBodyContains(
	t *testing.T,
	response *httptest.ResponseRecorder,
	values ...string,
) {
	t.Helper()

	body := response.Body.String()
	for _, want := range values {
		if !strings.Contains(body, want) {
			t.Fatalf("response body does not contain %q", want)
		}
	}
}

func TestPersonsPage(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personQueries.rows = []contacts.PersonRowView{
		{
			ID:       "101",
			Name:     "Anna Petrova",
			Position: "Head of Operations",
			Company:  "Northwind Logistics",
		},
	}

	response := fixture.get("/persons")
	assertStatus(t, response, http.StatusOK)

	contentType := response.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("GET /persons Content-Type = %q, want text/html", contentType)
	}

	assertBodyContains(
		t,
		response,
		"Position",
		"Anna Petrova",
		"Northwind Logistics",
		`href="/persons/new"`,
	)
}

func TestPersonsPageSearchAndPaging(t *testing.T) {
	fixture := newServerFixture(t)

	for i := 1; i <= rowsPerPage+1; i++ {
		fixture.personQueries.rows = append(
			fixture.personQueries.rows,
			contacts.PersonRowView{
				ID:       fmt.Sprintf("%d", i),
				Name:     fmt.Sprintf("Person %02d", i),
				Position: "Position",
			},
		)
	}

	response := fixture.get("/persons?q=anna&skip=20")
	assertStatus(t, response, http.StatusOK)

	filter := fixture.personQueries.filter
	if filter.Query != "anna" {
		t.Fatalf("filter.Query = %q, want %q", filter.Query, "anna")
	}
	if filter.Skip != 20 {
		t.Fatalf("filter.Skip = %d, want %d", filter.Skip, 20)
	}
	if filter.Limit != rowsPerPage+1 {
		t.Fatalf(
			"filter.Limit = %d, want %d",
			filter.Limit,
			rowsPerPage+1,
		)
	}

	assertBodyContains(
		t,
		response,
		`value="anna"`,
		`href="/persons?q=anna"`,
		`href="/persons?q=anna&amp;skip=40"`,
	)

	if strings.Contains(response.Body.String(), "Person 21") {
		t.Fatal("response contains lookahead row")
	}
}

func TestPersonsPageRejectsInvalidSkip(t *testing.T) {
	fixture := newServerFixture(t)

	for _, skip := range []string{"invalid", "-1"} {
		t.Run(skip, func(t *testing.T) {
			response := fixture.get("/persons?skip=" + skip)
			assertStatus(t, response, http.StatusBadRequest)
		})
	}
}

func TestPersonPages(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personQueries.person = contacts.PersonView{
		ID:       "101",
		Name:     "Anna Petrova",
		Position: "Engineer",
	}

	response := fixture.get("/persons/new")
	assertStatus(t, response, http.StatusOK)

	assertBodyContains(
		t,
		response,
		"New person",
		`action="/persons"`,
		"Create person",
	)

	if count := strings.Count(response.Body.String(), "<!doctype html>"); count != 1 {
		t.Fatalf("GET /persons/new document count = %d, want 1", count)
	}

	response = fixture.get("/persons/101?saved=1")
	assertStatus(t, response, http.StatusOK)
	if fixture.personQueries.id != common.ID(101) {
		t.Fatalf("GetPerson() id = %d, want 101", fixture.personQueries.id)
	}

	assertBodyContains(
		t,
		response,
		"Anna Petrova",
		"Engineer",
		`href="/persons/101/edit"`,
		"Person saved.",
		"Delete",
	)

	response = fixture.get("/persons/101/edit")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(
		t,
		response,
		"Edit person",
		`action="/persons/101"`,
		`value="Anna Petrova"`,
		`value="Engineer"`,
	)

	fixture.personQueries.err = contacts.ErrPersonNotFound
	response = fixture.get("/persons/999")
	assertStatus(t, response, http.StatusNotFound)
}

func TestPersonPagesRejectInvalidID(t *testing.T) {
	fixture := newServerFixture(t)
	values := url.Values{"name": {"Anna Petrova"}}

	for _, id := range []string{"0", "-1", "invalid"} {
		t.Run(id, func(t *testing.T) {
			getCalls := fixture.personQueries.getCalls
			response := fixture.get("/persons/" + id)
			assertStatus(t, response, http.StatusNotFound)
			if fixture.personQueries.getCalls != getCalls {
				t.Fatalf("GET /persons/%s reached GetPerson", id)
			}

			updateCalls := fixture.personCommands.updateCalls
			response = fixture.postForm("/persons/"+id, values)
			assertStatus(t, response, http.StatusNotFound)
			if fixture.personCommands.updateCalls != updateCalls {
				t.Fatalf("POST /persons/%s reached UpdatePerson", id)
			}
		})
	}
}

func TestPersonFormRejectsLargeBody(t *testing.T) {
	fixture := newServerFixture(t)

	tests := []struct {
		name  string
		path  string
		calls func() int
	}{
		{
			name: "create",
			path: "/persons",
			calls: func() int {
				return fixture.personCommands.createCalls
			},
		},
		{
			name: "update",
			path: "/persons/101",
			calls: func() int {
				return fixture.personCommands.updateCalls
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
			response := fixture.serve(request)
			assertStatus(t, response, http.StatusRequestEntityTooLarge)
			if tt.calls() != callsBefore {
				t.Fatalf("POST %s reached person command", tt.path)
			}
		})
	}
}

func TestCreatePerson(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personCommands.createID = common.ID(101)

	values := url.Values{
		"name":     {"  Anna Petrova  "},
		"position": {"  Engineer  "},
	}

	response := fixture.postForm("/persons", values)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?saved=1" {
		t.Fatalf("POST /persons Location = %q", location)
	}

	wantInput := contacts.PersonInput{
		Name:     "  Anna Petrova  ",
		Position: "  Engineer  ",
	}
	if fixture.personCommands.createInput != wantInput {
		t.Fatalf(
			"CreatePerson() input = %+v, want %+v",
			fixture.personCommands.createInput,
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
		fixture.personCommands.createErr = tt.err
		response = fixture.postForm("/persons", values)
		assertStatus(t, response, http.StatusUnprocessableEntity)
		assertBodyContains(
			t,
			response,
			tt.message,
			`value="  Anna Petrova  "`,
		)
	}

	fixture.personCommands.createErr = nil
	callsBefore := fixture.personCommands.createCalls

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

	response = fixture.serve(request)
	assertStatus(t, response, http.StatusForbidden)
	if fixture.personCommands.createCalls != callsBefore {
		t.Fatal("cross-origin POST reached CreatePerson")
	}
}

func TestUpdatePerson(t *testing.T) {
	fixture := newServerFixture(t)

	values := url.Values{
		"name":     {"  Anna Petrova  "},
		"position": {"  Director  "},
	}

	response := fixture.postForm("/persons/101", values)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?saved=1" {
		t.Fatalf("POST /persons/101 Location = %q", location)
	}

	if fixture.personCommands.updateID != common.ID(101) {
		t.Fatalf(
			"UpdatePerson() id = %d, want 101",
			fixture.personCommands.updateID,
		)
	}

	wantInput := contacts.PersonInput{
		Name:     "  Anna Petrova  ",
		Position: "  Director  ",
	}
	if fixture.personCommands.updateInput != wantInput {
		t.Fatalf(
			"UpdatePerson() input = %+v, want %+v",
			fixture.personCommands.updateInput,
			wantInput,
		)
	}

	fixture.personCommands.updateErr = contacts.ErrPersonNameExists
	response = fixture.postForm("/persons/101", values)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertBodyContains(
		t,
		response,
		"A person with this name already exists",
		`action="/persons/101"`,
		`value="  Anna Petrova  "`,
	)

	fixture.personCommands.updateErr = contacts.ErrPersonNotFound
	response = fixture.postForm("/persons/101", values)
	assertStatus(t, response, http.StatusNotFound)
}

func TestCompaniesPageSearchAndPaging(t *testing.T) {
	fixture := newServerFixture(t)

	for i := 1; i <= rowsPerPage+1; i++ {
		fixture.companyQueries.rows = append(
			fixture.companyQueries.rows,
			contacts.CompanyRowView{
				ID:      fmt.Sprintf("%d", i),
				Name:    fmt.Sprintf("Company %02d", i),
				Country: "Cyprus",
			},
		)
	}

	response := fixture.get("/companies?q=acme&skip=20")
	assertStatus(t, response, http.StatusOK)

	filter := fixture.companyQueries.filter
	if filter.Query != "acme" {
		t.Fatalf("filter.Query = %q, want %q", filter.Query, "acme")
	}
	if filter.Skip != 20 {
		t.Fatalf("filter.Skip = %d, want 20", filter.Skip)
	}
	if filter.Limit != rowsPerPage+1 {
		t.Fatalf(
			"filter.Limit = %d, want %d",
			filter.Limit,
			rowsPerPage+1,
		)
	}

	assertBodyContains(
		t,
		response,
		`value="acme"`,
		`href="/companies?q=acme"`,
		`href="/companies?q=acme&amp;skip=40"`,
		`href="/companies/1"`,
		"Company 01",
		"Cyprus",
	)

	if strings.Contains(response.Body.String(), "Company 21") {
		t.Fatal("response contains lookahead row")
	}
}

func TestCompanyPage(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.companyQueries.company = contacts.CompanyView{
		ID:      "101",
		Name:    "Northwind Logistics",
		Country: "Cyprus",
	}

	response := fixture.get("/companies/101?saved=1")
	assertStatus(t, response, http.StatusOK)
	if fixture.companyQueries.id != common.ID(101) {
		t.Fatalf("GetCompany() id = %d, want 101", fixture.companyQueries.id)
	}

	assertBodyContains(
		t,
		response,
		"Northwind Logistics",
		"Cyprus",
		"Company saved.",
		`href="/companies"`,
		`href="/companies/101/edit"`,
		`action="/companies/101/delete"`,
	)

	fixture.companyQueries.err = contacts.ErrCompanyNotFound
	response = fixture.get("/companies/999")
	assertStatus(t, response, http.StatusNotFound)
}

func TestCompanyPageRejectsInvalidID(t *testing.T) {
	fixture := newServerFixture(t)

	for _, id := range []string{"0", "-1", "invalid"} {
		t.Run(id, func(t *testing.T) {
			callsBefore := fixture.companyQueries.getCalls
			response := fixture.get("/companies/" + id)
			assertStatus(t, response, http.StatusNotFound)
			if fixture.companyQueries.getCalls != callsBefore {
				t.Fatalf("GET /companies/%s reached GetCompany", id)
			}
		})
	}
}

func TestNewAndCreateCompany(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.companyCommands.createID = common.ID(101)

	response := fixture.get("/companies/new")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(
		t,
		response,
		"New company",
		`action="/companies"`,
		"Create company",
		`name="country"`,
	)

	values := url.Values{
		"name":    {"  Northwind Logistics  "},
		"country": {"  Cyprus  "},
	}
	response = fixture.postForm("/companies", values)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/companies/101?saved=1" {
		t.Fatalf("POST /companies Location = %q", location)
	}

	wantInput := contacts.CompanyInput{
		Name:    "  Northwind Logistics  ",
		Country: "  Cyprus  ",
	}
	if fixture.companyCommands.createInput != wantInput {
		t.Fatalf(
			"CreateCompany() input = %+v, want %+v",
			fixture.companyCommands.createInput,
			wantInput,
		)
	}

	for _, tt := range []struct {
		err     error
		message string
	}{
		{
			err:     contacts.ErrCompanyNameRequired,
			message: "Name is required",
		},
		{
			err:     contacts.ErrCompanyNameExists,
			message: "A company with this name already exists",
		},
	} {
		fixture.companyCommands.createErr = tt.err
		response = fixture.postForm("/companies", values)
		assertStatus(t, response, http.StatusUnprocessableEntity)
		assertBodyContains(
			t,
			response,
			tt.message,
			`value="  Northwind Logistics  "`,
			`value="  Cyprus  "`,
		)
	}
}

func TestEditAndUpdateCompany(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.companyQueries.company = contacts.CompanyView{
		ID:      "101",
		Name:    "Northwind Logistics",
		Country: "Cyprus",
	}

	response := fixture.get("/companies/101/edit")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(
		t,
		response,
		"Edit company",
		`action="/companies/101"`,
		`value="Northwind Logistics"`,
		`value="Cyprus"`,
		"Save changes",
	)

	values := url.Values{
		"name":    {"  Northwind Group  "},
		"country": {"  France  "},
	}
	response = fixture.postForm("/companies/101", values)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/companies/101?saved=1" {
		t.Fatalf("POST /companies/101 Location = %q", location)
	}

	if fixture.companyCommands.updateID != common.ID(101) {
		t.Fatalf(
			"UpdateCompany() id = %d, want 101",
			fixture.companyCommands.updateID,
		)
	}

	wantInput := contacts.CompanyInput{
		Name:    "  Northwind Group  ",
		Country: "  France  ",
	}
	if fixture.companyCommands.updateInput != wantInput {
		t.Fatalf(
			"UpdateCompany() input = %+v, want %+v",
			fixture.companyCommands.updateInput,
			wantInput,
		)
	}

	fixture.companyCommands.updateErr = contacts.ErrCompanyNameExists
	response = fixture.postForm("/companies/101", values)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertBodyContains(
		t,
		response,
		"A company with this name already exists",
		`action="/companies/101"`,
		`value="  Northwind Group  "`,
		`value="  France  "`,
	)

	fixture.companyCommands.updateErr = contacts.ErrCompanyNotFound
	response = fixture.postForm("/companies/101", values)
	assertStatus(t, response, http.StatusNotFound)
}

func TestDeleteCompany(t *testing.T) {
	fixture := newServerFixture(t)

	response := fixture.postForm("/companies/101/delete", url.Values{})
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/companies" {
		t.Fatalf("delete Location = %q, want /companies", location)
	}
	if fixture.companyCommands.deleteID != common.ID(101) {
		t.Fatalf(
			"DeleteCompany() id = %d, want 101",
			fixture.companyCommands.deleteID,
		)
	}

	fixture.companyCommands.deleteErr = contacts.ErrCompanyNotFound
	response = fixture.postForm("/companies/101/delete", url.Values{})
	assertStatus(t, response, http.StatusNotFound)

	for _, id := range []string{"0", "-1", "invalid"} {
		t.Run(id, func(t *testing.T) {
			callsBefore := fixture.companyCommands.deleteCalls
			response := fixture.postForm(
				"/companies/"+id+"/delete",
				url.Values{},
			)
			assertStatus(t, response, http.StatusNotFound)
			if fixture.companyCommands.deleteCalls != callsBefore {
				t.Fatalf(
					"POST delete with id %s reached DeleteCompany",
					id,
				)
			}
		})
	}
}

func TestCompanyFormRejectsLargeBody(t *testing.T) {
	fixture := newServerFixture(t)

	tests := []struct {
		name  string
		path  string
		calls func() int
	}{
		{
			name: "create",
			path: "/companies",
			calls: func() int {
				return fixture.companyCommands.createCalls
			},
		},
		{
			name: "update",
			path: "/companies/101",
			calls: func() int {
				return fixture.companyCommands.updateCalls
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
					"name="+strings.Repeat(
						"a",
						maxCompanyFormBodySize,
					),
				),
			)
			request.Header.Set(
				"Content-Type",
				"application/x-www-form-urlencoded",
			)

			response := fixture.serve(request)
			assertStatus(t, response, http.StatusRequestEntityTooLarge)
			if tt.calls() != callsBefore {
				t.Fatalf("POST %s reached company command", tt.path)
			}
		})
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
		&recordingCompanyQueries{},
		&recordingCompanyCommands{},
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
