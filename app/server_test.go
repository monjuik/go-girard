package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/monjuik/go-girard/campaigns"
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

	deleteID    common.ID
	deleteErr   error
	deleteCalls int
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

type recordingEnrollmentQueries struct {
	personID   common.ID
	personRows []campaigns.PersonEnrollmentRowView
	err        error

	campaignCode  string
	campaignRows  []campaigns.CampaignPersonRowView
	campaignErr   error
	campaignCalls int

	dueThrough common.Date
	dueRows    []campaigns.DueEnrollmentRowView
	dueErr     error
}

type enrollmentTargetCall struct {
	PersonID     common.ID
	EnrollmentID common.ID
}

type postponeEnrollmentCall struct {
	enrollmentTargetCall
	Next common.Date
}

type moveEnrollmentCall struct {
	enrollmentTargetCall
	Step      string
	Next      common.Date
	Intention string
}

type recordingEnrollmentCommands struct {
	enrollPersonID common.ID
	enrollCampaign string
	enrollID       common.ID
	enrollErr      error
	enrollCalls    int

	postponeCall  postponeEnrollmentCall
	postponeErr   error
	postponeCalls int

	moveCall  moveEnrollmentCall
	moveErr   error
	moveCalls int

	completeCall  enrollmentTargetCall
	completeErr   error
	completeCalls int

	stopCall  enrollmentTargetCall
	stopErr   error
	stopCalls int
}

type serverFixture struct {
	handler            http.Handler
	campaigns          map[string]campaigns.Campaign
	personQueries      *recordingPersonQueries
	personCommands     *recordingPersonCommands
	companyQueries     *recordingCompanyQueries
	companyCommands    *recordingCompanyCommands
	enrollmentQueries  *recordingEnrollmentQueries
	enrollmentCommands *recordingEnrollmentCommands
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

func (c *recordingPersonCommands) DeletePerson(
	ctx context.Context,
	id common.ID,
) error {
	c.deleteCalls++
	c.deleteID = id
	return c.deleteErr
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

func (q *recordingEnrollmentQueries) ListPersonEnrollments(
	ctx context.Context,
	personID common.ID,
) ([]campaigns.PersonEnrollmentRowView, error) {
	q.personID = personID
	return q.personRows, q.err
}

func (q *recordingEnrollmentQueries) ListCampaignPersons(
	ctx context.Context,
	campaign string,
) ([]campaigns.CampaignPersonRowView, error) {
	q.campaignCalls++
	q.campaignCode = campaign
	return q.campaignRows, q.campaignErr
}

func (q *recordingEnrollmentQueries) ListDueEnrollments(
	ctx context.Context,
	through common.Date,
) ([]campaigns.DueEnrollmentRowView, error) {
	q.dueThrough = through
	return q.dueRows, q.dueErr
}

func (c *recordingEnrollmentCommands) Enroll(
	ctx context.Context,
	personID common.ID,
	campaignCode string,
) (common.ID, error) {
	c.enrollCalls++
	c.enrollPersonID = personID
	c.enrollCampaign = campaignCode
	return c.enrollID, c.enrollErr
}

func (c *recordingEnrollmentCommands) Postpone(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
	next common.Date,
) error {
	c.postponeCalls++
	c.postponeCall = postponeEnrollmentCall{
		enrollmentTargetCall: enrollmentTargetCall{
			PersonID:     personID,
			EnrollmentID: enrollmentID,
		},
		Next: next,
	}
	return c.postponeErr
}

func (c *recordingEnrollmentCommands) Move(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
	step string,
	next common.Date,
	intention string,
) error {
	c.moveCalls++
	c.moveCall = moveEnrollmentCall{
		enrollmentTargetCall: enrollmentTargetCall{
			PersonID:     personID,
			EnrollmentID: enrollmentID,
		},
		Step:      step,
		Next:      next,
		Intention: intention,
	}
	return c.moveErr
}

func (c *recordingEnrollmentCommands) Stop(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
) error {
	c.stopCalls++
	c.stopCall = enrollmentTargetCall{
		PersonID:     personID,
		EnrollmentID: enrollmentID,
	}
	return c.stopErr
}

func (c *recordingEnrollmentCommands) Complete(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
) error {
	c.completeCalls++
	c.completeCall = enrollmentTargetCall{
		PersonID:     personID,
		EnrollmentID: enrollmentID,
	}
	return c.completeErr
}

func newServerFixture(t *testing.T) *serverFixture {
	t.Helper()

	fixture := &serverFixture{
		campaigns:          make(map[string]campaigns.Campaign),
		personQueries:      &recordingPersonQueries{},
		personCommands:     &recordingPersonCommands{},
		companyQueries:     &recordingCompanyQueries{},
		companyCommands:    &recordingCompanyCommands{},
		enrollmentQueries:  &recordingEnrollmentQueries{},
		enrollmentCommands: &recordingEnrollmentCommands{},
	}

	server, err := NewServer(
		0,
		fixture.campaigns,
		fixture.personQueries,
		fixture.personCommands,
		fixture.companyQueries,
		fixture.companyCommands,
		fixture.enrollmentQueries,
		fixture.enrollmentCommands,
	)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	today, err := common.ParseDate("2026-09-03")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	server.today = func() common.Date { return today }

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

func TestDashboard(t *testing.T) {
	fixture := newServerFixture(t)
	campaign := newTestCampaign(t, campaigns.CampaignConfig{
		Code:    "followUp",
		Name:    "Follow-up",
		Version: 1,
		Steps: []campaigns.StepConfig{
			{Code: "restoreContext", Name: "Restore context"},
		},
	})
	fixture.campaigns[campaign.Code()] = campaign

	yesterday, err := common.ParseDate("2026-09-02")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	today, err := common.ParseDate("2026-09-03")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}
	fixture.enrollmentQueries.dueRows = []campaigns.DueEnrollmentRowView{
		{
			PersonID:   "101",
			PersonName: "Anna Petrova",
			Campaign:   "followUp",
			Next:       yesterday,
		},
		{
			PersonID:   "102",
			PersonName: "Boris Smirnov",
			Campaign:   "removedCampaign",
			Next:       today,
		},
	}

	response := fixture.get("/")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(
		t,
		response,
		"<title>Dashboard - Go Girard</title>",
		"<h1>Dashboard</h1>",
		"<h2>Due actions</h2>",
		`href="/"`,
		`aria-current="page"`,
		"<th>Name</th>",
		"<th>Campaign</th>",
		"<th>Date</th>",
		`href="/persons/101"`,
		"Anna Petrova",
		"Follow-up",
		`datetime="2026-09-02"`,
		`href="/persons/102"`,
		"Boris Smirnov",
		"removedCampaign",
		`datetime="2026-09-03"`,
	)

	body := response.Body.String()
	if strings.Index(body, `href="/"`) > strings.Index(body, `href="/persons"`) {
		t.Fatal("Dashboard is not the first menu item")
	}
	if strings.Index(body, "Anna Petrova") > strings.Index(body, "Boris Smirnov") {
		t.Fatal("Dashboard changed due enrollment order")
	}
	if fixture.enrollmentQueries.dueThrough != today {
		t.Fatalf(
			"ListDueEnrollments() through = %q, want %q",
			fixture.enrollmentQueries.dueThrough,
			today,
		)
	}

	fixture.enrollmentQueries.dueRows = nil
	response = fixture.get("/")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "No actions due.")

	fixture.enrollmentQueries.dueErr = fmt.Errorf("query failed")
	response = fixture.get("/")
	assertStatus(t, response, http.StatusInternalServerError)
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
	campaign := newTestCampaign(t, campaigns.CampaignConfig{
		Code:    "followUp",
		Name:    "Follow-up",
		Version: 1,
		Steps: []campaigns.StepConfig{
			{
				Code: "restoreContext",
				Name: "Restore context",
			},
			{
				Code: "discoverChanges",
				Name: "Discover changes",
			},
		},
	})
	fixture.campaigns[campaign.Code()] = campaign
	availableCampaign := newTestCampaign(t, campaigns.CampaignConfig{
		Code:    "businessDevelopment",
		Name:    "Business development",
		Version: 1,
		Steps: []campaigns.StepConfig{
			{
				Code: "introduction",
				Name: "Introduction",
			},
		},
	})
	fixture.campaigns[availableCampaign.Code()] = availableCampaign

	fixture.personQueries.person = contacts.PersonView{
		ID:          "101",
		Name:        "Anna Petrova",
		Position:    "Engineer",
		CompanyID:   "7",
		CompanyName: "Northwind Logistics",
		Note:        "## Responsibilities\n\n- **Operations**\n- [x] Reporting",
	}

	next, err := common.ParseDate("2026-09-02")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	fixture.enrollmentQueries.personRows =
		[]campaigns.PersonEnrollmentRowView{
			{
				ID:        "201",
				Campaign:  "followUp",
				Step:      "restoreContext",
				State:     campaigns.EnrollmentActive,
				Next:      next,
				Intention: "Restore context",
			},
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

	assertBodyContains(
		t,
		response,
		"<h2>Campaigns</h2>",
		"Follow-up",
		"<td>Active</td>",
		`datetime="2026-09-02"`,
		"Restore context",
		"Enroll…",
		`id="enroll_dialog"`,
		`action="/persons/101/enrollments"`,
		`value="businessDevelopment"`,
		"Business development",
		"<th>Actions</th>",
		"Postpone…",
		`id="postpone_dialog_201"`,
		`action="/persons/101/enrollments/201/postpone"`,
		"Move…",
		`id="move_dialog_201"`,
		`action="/persons/101/enrollments/201/move"`,
		`value="discoverChanges"`,
		`data-intention="Discover changes"`,
		`onchange="this.form.elements.intention.value = this.selectedOptions[0].dataset.intention"`,
		"Discover changes",
		"Complete…",
		`action="/persons/101/enrollments/201/complete"`,
		`onsubmit="return confirm('Complete this enrollment?')"`,
		"Stop…",
		`action="/persons/101/enrollments/201/stop"`,
		`onsubmit="return confirm('Stop this enrollment?')"`,
	)
	if strings.Contains(response.Body.String(), `value="followUp"`) {
		t.Fatal("enroll campaign options contain an existing enrollment")
	}
	if strings.Contains(response.Body.String(), `value="restoreContext"`) {
		t.Fatal("move step options contain the current step")
	}
	if strings.Contains(response.Body.String(), "Active ·") {
		t.Fatal("enrollment status contains the current step")
	}

	if fixture.enrollmentQueries.personID != common.ID(101) {
		t.Fatalf(
			"ListPersonEnrollments() person ID = %d, want 101",
			fixture.enrollmentQueries.personID,
		)
	}

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
		`action="/persons/101/delete"`,
		`class="grid"`,
		`href="/companies/7"`,
		"Northwind Logistics",
		`class="person-note"`,
		`<h2>Responsibilities</h2>`,
		`<strong>Operations</strong>`,
		`type="checkbox"`,
		`checked=""`,
	)

	response = fixture.get("/persons/101?enrolled=1")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "Person enrolled in campaign.")

	response = fixture.get("/persons/101?postponed=1")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "Action postponed.")

	response = fixture.get("/persons/101?moved=1")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "Enrollment moved.")

	response = fixture.get("/persons/101?completed=1")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "Enrollment completed.")

	response = fixture.get("/persons/101?stopped=1")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "Enrollment stopped.")

	fixture.enrollmentQueries.personRows[0].Step = "removedStep"
	response = fixture.get("/persons/101")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(
		t,
		response,
		"<td>Active</td>",
		`value="restoreContext"`,
		`value="discoverChanges"`,
	)
	if strings.Contains(response.Body.String(), "removedStep") {
		t.Fatal("removed enrollment step is displayed on the person page")
	}

	fixture.personQueries.person.CompanyID = ""
	fixture.personQueries.person.CompanyName = ""

	response = fixture.get("/persons/101")
	assertStatus(t, response, http.StatusOK)

	if strings.Contains(response.Body.String(), `href="/companies/7"`) {
		t.Fatal("person without company contains company link")
	}

	fixture.personQueries.person.CompanyID = "7"
	fixture.personQueries.person.CompanyName = "Northwind Logistics"

	response = fixture.get("/persons/101/edit")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(
		t,
		response,
		"Edit person",
		`action="/persons/101"`,
		`value="Anna Petrova"`,
		`value="Engineer"`,
		`name="company_name"`,
		`value="Northwind Logistics"`,
		`id="clear_company"`,
		`name="company_id"`,
		`value="7"`,
		`role="combobox"`,
		`aria-controls="company_options"`,
		`id="company_options"`,
		`role="listbox"`,
		`## Responsibilities`,
		`- **Operations**`,
		`- [x] Reporting`,
	)

	fixture.personQueries.person.Note = ""

	response = fixture.get("/persons/101")
	assertStatus(t, response, http.StatusOK)

	if strings.Contains(response.Body.String(), `class="person-note"`) {
		t.Fatal("person without note contains note section")
	}

	fixture.personQueries.err = contacts.ErrPersonNotFound
	response = fixture.get("/persons/999")
	assertStatus(t, response, http.StatusNotFound)
}

func TestEnrollPerson(t *testing.T) {
	fixture := newServerFixture(t)
	campaign := newTestCampaign(t, campaigns.CampaignConfig{
		Code:    "followUp",
		Name:    "Follow-up",
		Version: 1,
		Steps: []campaigns.StepConfig{
			{
				Code: "restoreContext",
				Name: "Restore context",
			},
		},
	})
	fixture.campaigns[campaign.Code()] = campaign
	fixture.personQueries.person = contacts.PersonView{
		ID:   "101",
		Name: "Anna Petrova",
	}
	fixture.enrollmentCommands.enrollID = common.ID(201)

	response := fixture.postForm(
		"/persons/101/enrollments",
		url.Values{"campaign": {"followUp"}},
	)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?enrolled=1" {
		t.Fatalf("enroll Location = %q, want person page", location)
	}
	if fixture.enrollmentCommands.enrollCalls != 1 {
		t.Fatalf(
			"Enroll() calls = %d, want 1",
			fixture.enrollmentCommands.enrollCalls,
		)
	}
	if fixture.enrollmentCommands.enrollPersonID != common.ID(101) {
		t.Fatalf(
			"Enroll() person ID = %d, want 101",
			fixture.enrollmentCommands.enrollPersonID,
		)
	}
	if fixture.enrollmentCommands.enrollCampaign != "followUp" {
		t.Fatalf(
			"Enroll() campaign = %q, want %q",
			fixture.enrollmentCommands.enrollCampaign,
			"followUp",
		)
	}

	fixture.enrollmentCommands.enrollErr = campaigns.ErrCampaignNotFound
	response = fixture.postForm(
		"/persons/101/enrollments",
		url.Values{"campaign": {"missingCampaign"}},
	)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	if fixture.enrollmentCommands.enrollCalls != 2 {
		t.Fatalf(
			"Enroll() calls = %d after unknown campaign, want 2",
			fixture.enrollmentCommands.enrollCalls,
		)
	}
	if fixture.enrollmentCommands.enrollCampaign != "missingCampaign" {
		t.Fatalf(
			"Enroll() campaign = %q, want missingCampaign",
			fixture.enrollmentCommands.enrollCampaign,
		)
	}

	fixture.enrollmentCommands.enrollErr = campaigns.ErrEnrollmentExists
	response = fixture.postForm(
		"/persons/101/enrollments",
		url.Values{"campaign": {"followUp"}},
	)
	assertStatus(t, response, http.StatusUnprocessableEntity)

	fixture.enrollmentCommands.enrollErr = nil
	fixture.personQueries.err = contacts.ErrPersonNotFound
	callsBefore := fixture.enrollmentCommands.enrollCalls
	response = fixture.postForm(
		"/persons/999/enrollments",
		url.Values{"campaign": {"followUp"}},
	)
	assertStatus(t, response, http.StatusNotFound)
	if fixture.enrollmentCommands.enrollCalls != callsBefore {
		t.Fatal("missing person reached Enroll()")
	}

	fixture.personQueries.err = nil
	request := httptest.NewRequest(
		http.MethodPost,
		"/persons/101/enrollments",
		strings.NewReader(
			"campaign="+strings.Repeat("a", maxEnrollmentFormBodySize),
		),
	)
	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)
	response = fixture.serve(request)
	assertStatus(t, response, http.StatusRequestEntityTooLarge)
}

func TestPostponeEnrollment(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personQueries.person = contacts.PersonView{
		ID:   "101",
		Name: "Anna Petrova",
	}

	next, err := common.ParseDate("2026-09-04")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	response := fixture.postForm(
		"/persons/101/enrollments/201/postpone",
		url.Values{"next": {next.String()}},
	)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?postponed=1" {
		t.Fatalf("postpone Location = %q, want person page", location)
	}
	if fixture.enrollmentCommands.postponeCalls != 1 {
		t.Fatalf(
			"Postpone() calls = %d, want 1",
			fixture.enrollmentCommands.postponeCalls,
		)
	}
	wantPostponeCall := postponeEnrollmentCall{
		enrollmentTargetCall: enrollmentTargetCall{
			PersonID:     common.ID(101),
			EnrollmentID: common.ID(201),
		},
		Next: next,
	}
	if fixture.enrollmentCommands.postponeCall != wantPostponeCall {
		t.Fatalf(
			"Postpone() call = %+v, want %+v",
			fixture.enrollmentCommands.postponeCall,
			wantPostponeCall,
		)
	}

	callsBefore := fixture.enrollmentCommands.postponeCalls
	response = fixture.postForm(
		"/persons/101/enrollments/201/postpone",
		url.Values{"next": {"invalid"}},
	)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	if fixture.enrollmentCommands.postponeCalls != callsBefore {
		t.Fatal("invalid date reached Postpone()")
	}

	for _, tt := range []struct {
		name       string
		commandErr error
		wantStatus int
	}{
		{"invalid next", campaigns.ErrEnrollmentNextInvalid, http.StatusUnprocessableEntity},
		{"date not future", campaigns.ErrEnrollmentDateNotFuture, http.StatusUnprocessableEntity},
		{"inactive", campaigns.ErrEnrollmentInactive, http.StatusUnprocessableEntity},
		{"not found", campaigns.ErrEnrollmentNotFound, http.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture.enrollmentCommands.postponeErr = tt.commandErr
			response := fixture.postForm(
				"/persons/101/enrollments/201/postpone",
				url.Values{"next": {next.String()}},
			)
			assertStatus(t, response, tt.wantStatus)
		})
	}

	fixture.enrollmentCommands.postponeErr = nil
	callsBefore = fixture.enrollmentCommands.postponeCalls
	response = fixture.postForm(
		"/persons/101/enrollments/invalid/postpone",
		url.Values{"next": {next.String()}},
	)
	assertStatus(t, response, http.StatusNotFound)
	if fixture.enrollmentCommands.postponeCalls != callsBefore {
		t.Fatal("invalid enrollment ID reached Postpone()")
	}
}

func TestMoveEnrollment(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personQueries.person = contacts.PersonView{
		ID:   "101",
		Name: "Anna Petrova",
	}

	next, err := common.ParseDate("2026-09-04")
	if err != nil {
		t.Fatalf("ParseDate() error = %v", err)
	}

	values := url.Values{
		"step":      {"discoverChanges"},
		"next":      {next.String()},
		"intention": {"Discover changes"},
	}
	response := fixture.postForm(
		"/persons/101/enrollments/201/move",
		values,
	)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?moved=1" {
		t.Fatalf("move Location = %q, want person page", location)
	}
	if fixture.enrollmentCommands.moveCalls != 1 {
		t.Fatalf(
			"Move() calls = %d, want 1",
			fixture.enrollmentCommands.moveCalls,
		)
	}
	wantMoveCall := moveEnrollmentCall{
		enrollmentTargetCall: enrollmentTargetCall{
			PersonID:     common.ID(101),
			EnrollmentID: common.ID(201),
		},
		Step:      "discoverChanges",
		Next:      next,
		Intention: "Discover changes",
	}
	if fixture.enrollmentCommands.moveCall != wantMoveCall {
		t.Fatalf(
			"Move() call = %+v, want %+v",
			fixture.enrollmentCommands.moveCall,
			wantMoveCall,
		)
	}

	callsBefore := fixture.enrollmentCommands.moveCalls
	invalidValues := url.Values{
		"step":      {"discoverChanges"},
		"next":      {"invalid"},
		"intention": {"Discover changes"},
	}
	response = fixture.postForm(
		"/persons/101/enrollments/201/move",
		invalidValues,
	)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	if fixture.enrollmentCommands.moveCalls != callsBefore {
		t.Fatal("invalid date reached Move()")
	}

	for _, tt := range []struct {
		name       string
		commandErr error
		wantStatus int
	}{
		{"campaign not found", campaigns.ErrCampaignNotFound, http.StatusUnprocessableEntity},
		{"invalid step", campaigns.ErrEnrollmentStepInvalid, http.StatusUnprocessableEntity},
		{"unchanged step", campaigns.ErrEnrollmentStepUnchanged, http.StatusUnprocessableEntity},
		{"invalid next", campaigns.ErrEnrollmentNextInvalid, http.StatusUnprocessableEntity},
		{"date not future", campaigns.ErrEnrollmentDateNotFuture, http.StatusUnprocessableEntity},
		{"intention required", campaigns.ErrEnrollmentIntentionRequired, http.StatusUnprocessableEntity},
		{"inactive", campaigns.ErrEnrollmentInactive, http.StatusUnprocessableEntity},
		{"not found", campaigns.ErrEnrollmentNotFound, http.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture.enrollmentCommands.moveErr = tt.commandErr
			response := fixture.postForm(
				"/persons/101/enrollments/201/move",
				values,
			)
			assertStatus(t, response, tt.wantStatus)
		})
	}

	fixture.enrollmentCommands.moveErr = nil
	callsBefore = fixture.enrollmentCommands.moveCalls
	response = fixture.postForm(
		"/persons/101/enrollments/invalid/move",
		values,
	)
	assertStatus(t, response, http.StatusNotFound)
	if fixture.enrollmentCommands.moveCalls != callsBefore {
		t.Fatal("invalid enrollment ID reached Move()")
	}
}

func TestCompleteEnrollment(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personQueries.person = contacts.PersonView{
		ID:   "101",
		Name: "Anna Petrova",
	}

	response := fixture.postForm(
		"/persons/101/enrollments/201/complete",
		nil,
	)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?completed=1" {
		t.Fatalf("complete Location = %q, want person page", location)
	}
	if fixture.enrollmentCommands.completeCalls != 1 {
		t.Fatalf(
			"Complete() calls = %d, want 1",
			fixture.enrollmentCommands.completeCalls,
		)
	}
	wantCompleteCall := enrollmentTargetCall{
		PersonID:     common.ID(101),
		EnrollmentID: common.ID(201),
	}
	if fixture.enrollmentCommands.completeCall != wantCompleteCall {
		t.Fatalf(
			"Complete() call = %+v, want %+v",
			fixture.enrollmentCommands.completeCall,
			wantCompleteCall,
		)
	}

	fixture.enrollmentCommands.completeErr = campaigns.ErrEnrollmentInactive
	response = fixture.postForm(
		"/persons/101/enrollments/201/complete",
		nil,
	)
	assertStatus(t, response, http.StatusUnprocessableEntity)

	fixture.enrollmentCommands.completeErr = campaigns.ErrEnrollmentNotFound
	response = fixture.postForm(
		"/persons/101/enrollments/201/complete",
		nil,
	)
	assertStatus(t, response, http.StatusNotFound)

	fixture.enrollmentCommands.completeErr = nil
	callsBefore := fixture.enrollmentCommands.completeCalls
	response = fixture.postForm(
		"/persons/101/enrollments/invalid/complete",
		nil,
	)
	assertStatus(t, response, http.StatusNotFound)
	if fixture.enrollmentCommands.completeCalls != callsBefore {
		t.Fatal("invalid enrollment ID reached Complete()")
	}
}

func TestStopEnrollment(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.personQueries.person = contacts.PersonView{
		ID:   "101",
		Name: "Anna Petrova",
	}

	response := fixture.postForm(
		"/persons/101/enrollments/201/stop",
		nil,
	)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?stopped=1" {
		t.Fatalf("stop Location = %q, want person page", location)
	}
	if fixture.enrollmentCommands.stopCalls != 1 {
		t.Fatalf(
			"Stop() calls = %d, want 1",
			fixture.enrollmentCommands.stopCalls,
		)
	}
	wantStopCall := enrollmentTargetCall{
		PersonID:     common.ID(101),
		EnrollmentID: common.ID(201),
	}
	if fixture.enrollmentCommands.stopCall != wantStopCall {
		t.Fatalf(
			"Stop() call = %+v, want %+v",
			fixture.enrollmentCommands.stopCall,
			wantStopCall,
		)
	}

	fixture.enrollmentCommands.stopErr = campaigns.ErrEnrollmentInactive
	response = fixture.postForm(
		"/persons/101/enrollments/201/stop",
		nil,
	)
	assertStatus(t, response, http.StatusUnprocessableEntity)

	fixture.enrollmentCommands.stopErr = campaigns.ErrEnrollmentNotFound
	response = fixture.postForm(
		"/persons/101/enrollments/201/stop",
		nil,
	)
	assertStatus(t, response, http.StatusNotFound)

	fixture.enrollmentCommands.stopErr = nil
	callsBefore := fixture.enrollmentCommands.stopCalls
	response = fixture.postForm(
		"/persons/101/enrollments/invalid/stop",
		nil,
	)
	assertStatus(t, response, http.StatusNotFound)
	if fixture.enrollmentCommands.stopCalls != callsBefore {
		t.Fatal("invalid enrollment ID reached Stop()")
	}
}

func TestInactiveEnrollmentHasNoActions(t *testing.T) {
	for _, state := range []campaigns.EnrollmentState{
		campaigns.EnrollmentCompleted,
		campaigns.EnrollmentStopped,
	} {
		t.Run(string(state), func(t *testing.T) {
			fixture := newServerFixture(t)
			fixture.personQueries.person = contacts.PersonView{
				ID:   "101",
				Name: "Anna Petrova",
			}
			fixture.enrollmentQueries.personRows = []campaigns.PersonEnrollmentRowView{
				{
					ID:        "201",
					Campaign:  "followUp",
					Step:      "restoreContext",
					State:     state,
					Intention: "Restore context",
				},
			}

			response := fixture.get("/persons/101")
			assertStatus(t, response, http.StatusOK)
			for _, action := range []string{
				"Postpone…",
				"Move…",
				"Complete…",
				"Stop…",
			} {
				if strings.Contains(response.Body.String(), action) {
					t.Fatalf("%s enrollment contains %s action", state, action)
				}
			}
		})
	}
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
		"name":         {"  Anna Petrova  "},
		"position":     {"  Engineer  "},
		"company_name": {"Northwind Logistics"},
		"company_id":   {"7"},
		"note":         {"## Responsibilities\n\n- Operations\n- Reporting"},
	}

	response := fixture.postForm("/persons", values)
	assertStatus(t, response, http.StatusSeeOther)
	if location := response.Header().Get("Location"); location != "/persons/101?saved=1" {
		t.Fatalf("POST /persons Location = %q", location)
	}

	wantInput := contacts.PersonInput{
		Name:      "  Anna Petrova  ",
		Position:  "  Engineer  ",
		CompanyID: common.ID(7),
		Note:      "## Responsibilities\n\n- Operations\n- Reporting",
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
			`value="Northwind Logistics"`,
		)
	}

	fixture.personCommands.createErr = contacts.ErrPersonCompanyNotFound

	response = fixture.postForm("/persons", values)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertBodyContains(
		t,
		response,
		"Selected company no longer exists",
		`value="Northwind Logistics"`,
	)

	fixture.personCommands.createErr = nil
	values.Set("company_id", "")
	callsBefore := fixture.personCommands.createCalls

	response = fixture.postForm("/persons", values)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertBodyContains(
		t,
		response,
		"Select a company from the list or clear the field",
		`value="Northwind Logistics"`,
	)
	if fixture.personCommands.createCalls != callsBefore {
		t.Fatal("unselected company reached CreatePerson")
	}

	values.Set("company_id", "7")

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
		"name":         {"  Anna Petrova  "},
		"position":     {"  Director  "},
		"company_name": {"Northwind Logistics"},
		"company_id":   {"7"},
		"note":         {"## Responsibilities\n\n- Operations\n- Reporting"},
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
		Name:      "  Anna Petrova  ",
		Position:  "  Director  ",
		CompanyID: common.ID(7),
		Note:      "## Responsibilities\n\n- Operations\n- Reporting",
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
		`value="Northwind Logistics"`,
	)

	fixture.personCommands.updateErr = nil
	values.Set("company_id", "")
	callsBefore := fixture.personCommands.updateCalls

	response = fixture.postForm("/persons/101", values)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertBodyContains(
		t,
		response,
		"Select a company from the list or clear the field",
		`value="Northwind Logistics"`,
	)

	if fixture.personCommands.updateCalls != callsBefore {
		t.Fatal("unselected company reached UpdatePerson")
	}

	values.Set("company_id", "7")

	fixture.personCommands.updateErr = contacts.ErrPersonNotFound
	response = fixture.postForm("/persons/101", values)
	assertStatus(t, response, http.StatusNotFound)

	fixture.personCommands.updateErr = contacts.ErrPersonCompanyNotFound
	response = fixture.postForm("/persons/101", values)
	assertStatus(t, response, http.StatusUnprocessableEntity)
	assertBodyContains(
		t,
		response,
		"Selected company no longer exists",
		`action="/persons/101"`,
		`value="Northwind Logistics"`,
	)

	fixture.personCommands.updateErr = nil
	callsBefore = fixture.personCommands.updateCalls

	values.Set("company_id", "invalid")
	response = fixture.postForm("/persons/101", values)

	assertStatus(t, response, http.StatusBadRequest)

	if fixture.personCommands.updateCalls != callsBefore {
		t.Fatal("invalid company ID reached UpdatePerson")
	}
}

func TestDeletePerson(t *testing.T) {
	fixture := newServerFixture(t)

	response := fixture.postForm("/persons/101/delete", url.Values{})
	assertStatus(t, response, http.StatusSeeOther)

	if location := response.Header().Get("Location"); location != "/persons" {
		t.Fatalf("delete Location = %q, want /persons", location)
	}
	if fixture.personCommands.deleteID != common.ID(101) {
		t.Fatalf(
			"DeletePerson() id = %d, want 101",
			fixture.personCommands.deleteID,
		)
	}

	fixture.personCommands.deleteErr = contacts.ErrPersonNotFound
	response = fixture.postForm("/persons/101/delete", url.Values{})
	assertStatus(t, response, http.StatusNotFound)

	for _, id := range []string{"0", "-1", "invalid"} {
		t.Run(id, func(t *testing.T) {
			callsBefore := fixture.personCommands.deleteCalls

			response := fixture.postForm(
				"/persons/"+id+"/delete",
				url.Values{},
			)
			assertStatus(t, response, http.StatusNotFound)

			if fixture.personCommands.deleteCalls != callsBefore {
				t.Fatalf(
					"POST delete with id %s reached DeletePerson",
					id,
				)
			}
		})
	}
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
		nil,
		&recordingPersonQueries{},
		commands,
		&recordingCompanyQueries{},
		&recordingCompanyCommands{},
		nil,
		nil,
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

func TestCompaniesJSONSearch(t *testing.T) {
	fixture := newServerFixture(t)
	fixture.companyQueries.rows = []contacts.CompanyRowView{
		{
			ID:      "101",
			Name:    "Northwind Logistics",
			Country: "Cyprus",
		},
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/companies?q=north&skip=40",
		nil,
	)
	request.Header.Set("Accept", "application/json")

	response := fixture.serve(request)
	assertStatus(t, response, http.StatusOK)

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf(
			"Content-Type = %q, want application/json",
			contentType,
		)
	}

	filter := fixture.companyQueries.filter
	if filter.Query != "north" {
		t.Fatalf("filter.Query = %q, want north", filter.Query)
	}
	if filter.Skip != 0 {
		t.Fatalf("filter.Skip = %d, want 0", filter.Skip)
	}
	if filter.Limit != rowsPerPage {
		t.Fatalf(
			"filter.Limit = %d, want %d",
			filter.Limit,
			rowsPerPage,
		)
	}

	wantBody := `[{"id":"101","name":"Northwind Logistics"}]`
	if body := response.Body.String(); body != wantBody {
		t.Fatalf("response body = %q, want %q", body, wantBody)
	}
}

func TestConfigPage(t *testing.T) {
	fixture := newServerFixture(t)

	for _, config := range []campaigns.CampaignConfig{
		{
			Code:    "followUp",
			Name:    "Follow-up",
			Version: 1,
			Steps: []campaigns.StepConfig{
				{Code: "contact", Name: "Contact"},
			},
		},
		{
			Code:    "businessDevelopment",
			Name:    "Business development",
			Version: 1,
			Steps: []campaigns.StepConfig{
				{Code: "discoverPlans", Name: "Discover plans"},
			},
		},
	} {
		campaign := newTestCampaign(t, config)
		fixture.campaigns[campaign.Code()] = campaign
	}

	response := fixture.get("/config/")
	assertStatus(t, response, http.StatusOK)

	assertBodyContains(
		t,
		response,
		"<h1>Campaigns</h1>",
		`href="/config/campaigns/businessDevelopment"`,
		`href="/config/campaigns/followUp"`,
		`href="/config/"`,
		`aria-current="page"`,
	)

	body := response.Body.String()
	if strings.Index(body, "businessDevelopment") >
		strings.Index(body, "followUp") {
		t.Fatal("campaigns are not sorted by code")
	}
}

func TestCampaignPage(t *testing.T) {
	fixture := newServerFixture(t)

	campaign := newTestCampaign(t, campaigns.CampaignConfig{
		Code:        "followUp",
		Name:        "Follow-up",
		Version:     2,
		Description: "Resume **earlier discussions**.",
		Steps: []campaigns.StepConfig{
			{
				Code:         "restoreContext",
				Name:         "Restore context",
				Instructions: "Review the *previous discussion*.",
			},
			{
				Code: "agreeNextAction",
				Name: "Agree next action",
			},
		},
	})
	fixture.campaigns[campaign.Code()] = campaign
	fixture.enrollmentQueries.campaignRows = []campaigns.CampaignPersonRowView{
		{
			PersonID:   "101",
			PersonName: "Anna Petrova",
			CompanyID:  "7",
			Company:    "Northwind Logistics",
			State:      campaigns.EnrollmentActive,
		},
		{
			PersonID:   "102",
			PersonName: "Boris Smirnov",
			State:      campaigns.EnrollmentCompleted,
		},
		{
			PersonID:   "103",
			PersonName: "Carla Gomez",
			State:      campaigns.EnrollmentStopped,
		},
	}

	response := fixture.get("/config/campaigns/followUp")
	assertStatus(t, response, http.StatusOK)

	assertBodyContains(
		t,
		response,
		"<h1>Follow-up</h1>",
		"<strong>earlier discussions</strong>",
		"<em>previous discussion</em>",
		"<code>followUp</code>",
		"<code>restoreContext</code>",
		"<code>agreeNextAction</code>",
		"<h2>Persons</h2>",
		"<th>Name</th>",
		"<th>Company</th>",
		"<th>Status</th>",
		`href="/persons/101"`,
		"Anna Petrova",
		`href="/companies/7"`,
		"Northwind Logistics",
		"<td>Active</td>",
		`href="/persons/102"`,
		"Boris Smirnov",
		"<td>Completed</td>",
		`href="/persons/103"`,
		"Carla Gomez",
		"<td>Stopped</td>",
		"&mdash;",
	)
	if fixture.enrollmentQueries.campaignCode != "followUp" {
		t.Fatalf(
			"ListCampaignPersons() campaign = %q, want followUp",
			fixture.enrollmentQueries.campaignCode,
		)
	}

	body := response.Body.String()
	if strings.Index(body, "restoreContext") >
		strings.Index(body, "agreeNextAction") {
		t.Fatal("campaign steps are not in configured order")
	}
	if strings.Index(body, "Anna Petrova") >
		strings.Index(body, "Boris Smirnov") ||
		strings.Index(body, "Boris Smirnov") >
			strings.Index(body, "Carla Gomez") {
		t.Fatal("campaign page changed person order")
	}

	fixture.enrollmentQueries.campaignRows = nil
	response = fixture.get("/config/campaigns/followUp")
	assertStatus(t, response, http.StatusOK)
	assertBodyContains(t, response, "No persons enrolled.")

	fixture.enrollmentQueries.campaignErr = fmt.Errorf("query failed")
	response = fixture.get("/config/campaigns/followUp")
	assertStatus(t, response, http.StatusInternalServerError)

	callsBefore := fixture.enrollmentQueries.campaignCalls
	response = fixture.get("/config/campaigns/missing")
	assertStatus(t, response, http.StatusNotFound)
	if fixture.enrollmentQueries.campaignCalls != callsBefore {
		t.Fatal("missing campaign reached ListCampaignPersons()")
	}
}

func newTestCampaign(
	t *testing.T,
	config campaigns.CampaignConfig,
) campaigns.Campaign {
	t.Helper()

	campaign, err := campaigns.NewCampaign(config)
	if err != nil {
		t.Fatalf("NewCampaign() error = %v", err)
	}
	return campaign
}
