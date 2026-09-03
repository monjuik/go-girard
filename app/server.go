package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/monjuik/go-girard/campaigns"
	"github.com/monjuik/go-girard/common"
	"github.com/monjuik/go-girard/contacts"
)

const (
	rowsPerPage               = 20
	maxPersonFormBodySize     = 64 << 10
	maxCompanyFormBodySize    = 64 << 10
	maxEnrollmentFormBodySize = 8 << 10
)

type PersonsPageData struct {
	Persons     []contacts.PersonRowView
	Query       string
	PreviousURL string
	NextURL     string
}

type DashboardPageData struct {
	Enrollments []DashboardEnrollmentPageData
}

type DashboardEnrollmentPageData struct {
	PersonID   string
	PersonName string
	Campaign   string
	Next       string
}

// PersonPageData contains data for the read-only person page.
type PersonPageData struct {
	Person             contacts.PersonView
	Enrollments        []PersonEnrollmentPageData
	AvailableCampaigns []campaigns.CampaignRowView
	Note               template.HTML
	Saved              bool
	Enrolled           bool
	Postponed          bool
	Moved              bool
	Completed          bool
	Stopped            bool
}

type PersonEnrollmentPageData struct {
	ID           string
	CampaignCode string
	CampaignName string
	Status       string
	Next         string
	Intention    string
	Active       bool
	MoveSteps    []campaigns.Step
}

// PersonFormData contains data for the create and edit form.
type PersonFormData struct {
	Heading      string
	Action       string
	SubmitLabel  string
	Input        contacts.PersonInput
	CompanyName  string
	NameError    string
	CompanyError string
}

type CompaniesPageData struct {
	Companies   []contacts.CompanyRowView
	Query       string
	PreviousURL string
	NextURL     string
}

type companyOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CompanyPageData contains data for the read-only conpany page.
type CompanyPageData struct {
	Company contacts.CompanyView
	Saved   bool
}

// CompanyFormData contains data for the create and edit form.
type CompanyFormData struct {
	Heading     string
	Action      string
	SubmitLabel string
	Input       contacts.CompanyInput
	NameError   string
}

type CampaignsPageData struct {
	Campaigns []campaigns.CampaignRowView
}

type CampaignPageData struct {
	Campaign    campaigns.Campaign
	Description template.HTML
	Steps       []CampaignStepPageData
	Persons     []CampaignPersonPageData
}

type CampaignStepPageData struct {
	Step         campaigns.Step
	Instructions template.HTML
}

type CampaignPersonPageData struct {
	PersonID   string
	PersonName string
	CompanyID  string
	Company    string
	Status     string
}

type Server struct {
	campaigns          map[string]campaigns.Campaign
	personQueries      contacts.PersonQueries
	personCommands     contacts.PersonCommands
	companyQueries     contacts.CompanyQueries
	companyCommands    contacts.CompanyCommands
	enrollmentQueries  campaigns.EnrollmentQueries
	enrollmentCommands campaigns.EnrollmentCommands
	today              func() common.Date
	templates          *Templates
	httpServer         *http.Server
}

func NewServer(
	port int,
	campaigns map[string]campaigns.Campaign,
	personQueries contacts.PersonQueries,
	personCommands contacts.PersonCommands,
	companyQueries contacts.CompanyQueries,
	companyCommands contacts.CompanyCommands,
	enrollmentQueries campaigns.EnrollmentQueries,
	enrollmentCommands campaigns.EnrollmentCommands,
) (*Server, error) {
	templates, err := NewTemplates()
	if err != nil {
		return nil, err
	}
	server := &Server{
		campaigns:          campaigns,
		personQueries:      personQueries,
		personCommands:     personCommands,
		companyQueries:     companyQueries,
		companyCommands:    companyCommands,
		enrollmentQueries:  enrollmentQueries,
		enrollmentCommands: enrollmentCommands,
		today: func() common.Date {
			return common.DateFromTime(time.Now())
		},
		templates: templates,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", server.handleDashboard)
	mux.HandleFunc("GET /persons", server.handlePersons)
	mux.HandleFunc("GET /persons/new", server.handleNewPerson)
	mux.HandleFunc("GET /persons/{id}", server.handlePerson)
	mux.HandleFunc("GET /persons/{id}/edit", server.handleEditPerson)
	mux.HandleFunc("GET /companies", server.handleCompanies)
	mux.HandleFunc("GET /companies/new", server.handleNewCompany)
	mux.HandleFunc("GET /companies/{id}", server.handleCompany)
	mux.HandleFunc("GET /companies/{id}/edit", server.handleEditCompany)
	mux.HandleFunc("GET /config/{$}", server.handleConfig)
	mux.HandleFunc("GET /config/campaigns/{code}", server.handleCampaign)
	mux.HandleFunc("POST /companies", server.handleCreateCompany)
	mux.HandleFunc("POST /companies/{id}", server.handleUpdateCompany)
	mux.HandleFunc("POST /companies/{id}/delete", server.handleDeleteCompany)
	mux.HandleFunc("POST /persons", server.handleCreatePerson)
	mux.HandleFunc("POST /persons/{id}", server.handleUpdatePerson)
	mux.HandleFunc("POST /persons/{id}/delete", server.handleDeletePerson)
	mux.HandleFunc("POST /persons/{id}/enrollments", server.handleEnrollPerson)
	mux.HandleFunc(
		"POST /persons/{id}/enrollments/{enrollmentID}/postpone",
		server.handlePostponeEnrollment,
	)
	mux.HandleFunc(
		"POST /persons/{id}/enrollments/{enrollmentID}/move",
		server.handleMoveEnrollment,
	)
	mux.HandleFunc(
		"POST /persons/{id}/enrollments/{enrollmentID}/complete",
		server.handleCompleteEnrollment,
	)
	mux.HandleFunc(
		"POST /persons/{id}/enrollments/{enrollmentID}/stop",
		server.handleStopEnrollment,
	)
	addr := fmt.Sprintf(":%d", port)
	protection := http.NewCrossOriginProtection()
	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: protection.Handler(mux),
	}
	return server, nil
}

func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Addr() string {
	return s.httpServer.Addr
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	rows, err := s.enrollmentQueries.ListDueEnrollments(
		r.Context(),
		s.today(),
	)
	if err != nil {
		http.Error(w, "failed to list due enrollments", http.StatusInternalServerError)
		return
	}

	enrollments := make([]DashboardEnrollmentPageData, 0, len(rows))
	for _, row := range rows {
		campaignName := row.Campaign
		if campaign, exists := s.campaigns[row.Campaign]; exists {
			campaignName = campaign.Name()
		}

		enrollments = append(enrollments, DashboardEnrollmentPageData{
			PersonID:   row.PersonID,
			PersonName: row.PersonName,
			Campaign:   campaignName,
			Next:       row.Next.String(),
		})
	}

	s.templates.Render(w, "dashboard", PageData{
		Title:      "Dashboard",
		ActiveMenu: "dashboard",
		Data: DashboardPageData{
			Enrollments: enrollments,
		},
	})
}

func (s *Server) handlePersons(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	query := values.Get("q")

	skip, err := parseSkip(values.Get("skip"))
	if err != nil {
		http.Error(w, "invalid skip", http.StatusBadRequest)
		return
	}

	persons, err := s.personQueries.ListPersonRows(
		r.Context(),
		contacts.PersonsFilter{
			Query: query,
			Skip:  skip,
			Limit: rowsPerPage + 1,
		},
	)
	if err != nil {
		http.Error(w, "failed to list persons", http.StatusInternalServerError)
		return
	}
	hasNext := len(persons) > rowsPerPage
	if hasNext {
		persons = persons[:rowsPerPage]
	}
	data := PersonsPageData{
		Persons: persons,
		Query:   query,
	}

	if skip > 0 {
		data.PreviousURL = buildPersonsURL(
			query,
			max(0, skip-rowsPerPage),
		)
	}

	if hasNext {
		data.NextURL = buildPersonsURL(
			query,
			skip+rowsPerPage,
		)
	}

	s.templates.Render(w, "persons", PageData{
		Title:      "Persons",
		ActiveMenu: "persons",
		Data:       data,
	})
}

func (s *Server) handleNewPerson(w http.ResponseWriter, r *http.Request) {
	s.renderPersonForm(w, PersonFormData{
		Heading:     "New person",
		Action:      "/persons",
		SubmitLabel: "Create person",
	}, http.StatusOK)
}

func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	personID, err := common.IDFromString(person.ID)
	if err != nil {
		http.Error(
			w,
			"failed to get person enrollments",
			http.StatusInternalServerError,
		)
		return
	}

	enrollmentRows, err := s.enrollmentQueries.ListPersonEnrollments(
		r.Context(),
		personID,
	)
	if err != nil {
		http.Error(
			w,
			"failed to list person enrollments",
			http.StatusInternalServerError,
		)
		return
	}

	enrollments := s.personEnrollmentPageData(enrollmentRows)
	availableCampaigns := s.availableCampaigns(enrollmentRows)

	noteHTML, err := renderMarkdown(person.Note)
	if err != nil {
		noteHTML = "<p>Error: markdown engine failed to render person note</p>"
	}

	s.templates.Render(w, "person", PageData{
		Title:      person.Name,
		ActiveMenu: "persons",
		Data: PersonPageData{
			Person:             person,
			Enrollments:        enrollments,
			AvailableCampaigns: availableCampaigns,
			Note:               noteHTML,
			Saved:              r.URL.Query().Get("saved") == "1",
			Enrolled:           r.URL.Query().Get("enrolled") == "1",
			Postponed:          r.URL.Query().Get("postponed") == "1",
			Moved:              r.URL.Query().Get("moved") == "1",
			Completed:          r.URL.Query().Get("completed") == "1",
			Stopped:            r.URL.Query().Get("stopped") == "1",
		},
	})
}

func (s *Server) handleEnrollPerson(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	personID, err := common.IDFromString(person.ID)
	if err != nil {
		http.Error(w, "failed to enroll person", http.StatusInternalServerError)
		return
	}

	if err := parseEnrollmentForm(w, r); err != nil {
		writeEnrollmentFormError(w, err)
		return
	}

	campaignCode := r.PostForm.Get("campaign")
	_, err = s.enrollmentCommands.Enroll(
		r.Context(),
		personID,
		campaignCode,
	)
	switch {
	case errors.Is(err, campaigns.ErrCampaignNotFound):
		http.Error(w, "select a configured campaign", http.StatusUnprocessableEntity)
		return
	case errors.Is(err, campaigns.ErrEnrollmentExists):
		http.Error(w, "person is already enrolled in this campaign", http.StatusUnprocessableEntity)
		return
	case errors.Is(err, campaigns.ErrEnrollmentPersonNotFound):
		http.NotFound(w, r)
		return
	case err != nil:
		http.Error(w, "failed to enroll person", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/persons/"+person.ID+"?enrolled=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handlePostponeEnrollment(
	w http.ResponseWriter,
	r *http.Request,
) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	personID, err := common.IDFromString(person.ID)
	if err != nil {
		http.Error(w, "failed to postpone enrollment", http.StatusInternalServerError)
		return
	}

	enrollmentID, err := common.IDFromString(r.PathValue("enrollmentID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := parseEnrollmentForm(w, r); err != nil {
		writeEnrollmentFormError(w, err)
		return
	}

	next, err := common.ParseDate(r.PostForm.Get("next"))
	if err != nil {
		http.Error(w, "select a valid date", http.StatusUnprocessableEntity)
		return
	}

	err = s.enrollmentCommands.Postpone(
		r.Context(),
		personID,
		enrollmentID,
		next,
	)
	switch {
	case errors.Is(err, campaigns.ErrEnrollmentNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, campaigns.ErrEnrollmentNextInvalid),
		errors.Is(err, campaigns.ErrEnrollmentDateNotFuture),
		errors.Is(err, campaigns.ErrEnrollmentInactive):
		http.Error(
			w,
			"select a future date for an active enrollment",
			http.StatusUnprocessableEntity,
		)
		return
	case err != nil:
		http.Error(w, "failed to postpone enrollment", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/persons/"+person.ID+"?postponed=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleMoveEnrollment(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	personID, err := common.IDFromString(person.ID)
	if err != nil {
		http.Error(w, "failed to move enrollment", http.StatusInternalServerError)
		return
	}

	enrollmentID, err := common.IDFromString(r.PathValue("enrollmentID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := parseEnrollmentForm(w, r); err != nil {
		writeEnrollmentFormError(w, err)
		return
	}

	next, err := common.ParseDate(r.PostForm.Get("next"))
	if err != nil {
		http.Error(w, "select a valid date", http.StatusUnprocessableEntity)
		return
	}

	err = s.enrollmentCommands.Move(
		r.Context(),
		personID,
		enrollmentID,
		r.PostForm.Get("step"),
		next,
		r.PostForm.Get("intention"),
	)
	switch {
	case errors.Is(err, campaigns.ErrEnrollmentNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, campaigns.ErrCampaignNotFound),
		errors.Is(err, campaigns.ErrEnrollmentStepInvalid),
		errors.Is(err, campaigns.ErrEnrollmentStepUnchanged),
		errors.Is(err, campaigns.ErrEnrollmentNextInvalid),
		errors.Is(err, campaigns.ErrEnrollmentDateNotFuture),
		errors.Is(err, campaigns.ErrEnrollmentIntentionRequired),
		errors.Is(err, campaigns.ErrEnrollmentInactive):
		http.Error(
			w,
			"select a different configured step, a future date, and an intention",
			http.StatusUnprocessableEntity,
		)
		return
	case err != nil:
		http.Error(w, "failed to move enrollment", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/persons/"+person.ID+"?moved=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleStopEnrollment(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	personID, err := common.IDFromString(person.ID)
	if err != nil {
		http.Error(w, "failed to stop enrollment", http.StatusInternalServerError)
		return
	}

	enrollmentID, err := common.IDFromString(r.PathValue("enrollmentID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = s.enrollmentCommands.Stop(
		r.Context(),
		personID,
		enrollmentID,
	)
	switch {
	case errors.Is(err, campaigns.ErrEnrollmentNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, campaigns.ErrEnrollmentInactive):
		http.Error(w, "enrollment is not active", http.StatusUnprocessableEntity)
		return
	case err != nil:
		http.Error(w, "failed to stop enrollment", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/persons/"+person.ID+"?stopped=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleCompleteEnrollment(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	personID, err := common.IDFromString(person.ID)
	if err != nil {
		http.Error(w, "failed to complete enrollment", http.StatusInternalServerError)
		return
	}

	enrollmentID, err := common.IDFromString(r.PathValue("enrollmentID"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = s.enrollmentCommands.Complete(
		r.Context(),
		personID,
		enrollmentID,
	)
	switch {
	case errors.Is(err, campaigns.ErrEnrollmentNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, campaigns.ErrEnrollmentInactive):
		http.Error(w, "enrollment is not active", http.StatusUnprocessableEntity)
		return
	case err != nil:
		http.Error(w, "failed to complete enrollment", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/persons/"+person.ID+"?completed=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleEditPerson(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}
	input := contacts.PersonInput{
		Name:     person.Name,
		Position: person.Position,
		Note:     person.Note,
	}

	if person.CompanyID != "" {
		companyID, err := common.IDFromString(person.CompanyID)
		if err != nil {
			http.Error(w, "failed to get person", http.StatusInternalServerError)
			return
		}

		input.CompanyID = companyID
	}

	s.renderPersonForm(w, PersonFormData{
		Heading:     "Edit person",
		Action:      "/persons/" + person.ID,
		SubmitLabel: "Save changes",
		Input:       input,
		CompanyName: person.CompanyName,
	}, http.StatusOK)
}

func (s *Server) handleCreatePerson(w http.ResponseWriter, r *http.Request) {
	input, err := parsePersonInput(w, r)
	if err != nil {
		writePersonFormError(w, err)
		return
	}

	companyName := r.PostForm.Get("company_name")

	if companyError := personCompanySelectionError(input, companyName); companyError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:      "New person",
			Action:       "/persons",
			SubmitLabel:  "Create person",
			Input:        input,
			CompanyName:  companyName,
			CompanyError: companyError,
		}, http.StatusUnprocessableEntity)
		return
	}

	id, err := s.personCommands.CreatePerson(r.Context(), input)
	if nameError := personNameError(err); nameError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:     "New person",
			Action:      "/persons",
			SubmitLabel: "Create person",
			Input:       input,
			NameError:   nameError,
			CompanyName: companyName,
		}, http.StatusUnprocessableEntity)
		return
	}

	if companyError := personCompanyError(err); companyError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:      "New person",
			Action:       "/persons",
			SubmitLabel:  "Create person",
			Input:        input,
			CompanyName:  companyName,
			CompanyError: companyError,
		}, http.StatusUnprocessableEntity)
		return
	}

	if err != nil {
		http.Error(w, "failed to create person", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/persons/"+id.String()+"?saved=1", http.StatusSeeOther)
}

func (s *Server) handleUpdatePerson(w http.ResponseWriter, r *http.Request) {
	id, err := common.IDFromString(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	input, err := parsePersonInput(w, r)
	if err != nil {
		writePersonFormError(w, err)
		return
	}

	companyName := r.PostForm.Get("company_name")

	if companyError := personCompanySelectionError(
		input,
		companyName,
	); companyError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:      "Edit person",
			Action:       "/persons/" + id.String(),
			SubmitLabel:  "Save changes",
			Input:        input,
			CompanyName:  companyName,
			CompanyError: companyError,
		}, http.StatusUnprocessableEntity)
		return
	}

	err = s.personCommands.UpdatePerson(r.Context(), id, input)
	if nameError := personNameError(err); nameError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:     "Edit person",
			Action:      "/persons/" + id.String(),
			SubmitLabel: "Save changes",
			Input:       input,
			NameError:   nameError,
			CompanyName: companyName,
		}, http.StatusUnprocessableEntity)
		return
	}
	if errors.Is(err, contacts.ErrPersonNotFound) {
		http.NotFound(w, r)
		return
	}

	if companyError := personCompanyError(err); companyError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:      "Edit person",
			Action:       "/persons/" + id.String(),
			SubmitLabel:  "Save changes",
			Input:        input,
			CompanyName:  companyName,
			CompanyError: companyError,
		}, http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, "failed to update person", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/persons/"+id.String()+"?saved=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleDeletePerson(w http.ResponseWriter, r *http.Request) {
	id, err := common.IDFromString(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = s.personCommands.DeletePerson(r.Context(), id)
	if errors.Is(err, contacts.ErrPersonNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to delete person", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/persons", http.StatusSeeOther)
}

func (s *Server) handleCompanies(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == "application/json" {
		s.handleCompanyOptions(w, r)
		return
	}

	values := r.URL.Query()
	query := values.Get("q")

	skip, err := parseSkip(values.Get("skip"))
	if err != nil {
		http.Error(w, "invalid skip", http.StatusBadRequest)
		return
	}

	companies, err := s.companyQueries.ListCompanyRows(
		r.Context(),
		contacts.CompaniesFilter{
			Query: query,
			Skip:  skip,
			Limit: rowsPerPage + 1,
		},
	)
	if err != nil {
		http.Error(w, "failed to list companies", http.StatusInternalServerError)
		return
	}

	hasNext := len(companies) > rowsPerPage
	if hasNext {
		companies = companies[:rowsPerPage]
	}

	data := CompaniesPageData{
		Companies: companies,
		Query:     query,
	}

	if skip > 0 {
		data.PreviousURL = buildCompaniesURL(
			query,
			max(0, skip-rowsPerPage),
		)
	}

	if hasNext {
		data.NextURL = buildCompaniesURL(
			query,
			skip+rowsPerPage,
		)
	}

	s.templates.Render(w, "companies", PageData{
		Title:      "Companies",
		ActiveMenu: "companies",
		Data:       data,
	})
}

func (s *Server) handleCompanyOptions(
	w http.ResponseWriter,
	r *http.Request,
) {
	rows, err := s.companyQueries.ListCompanyRows(
		r.Context(),
		contacts.CompaniesFilter{
			Query: r.URL.Query().Get("q"),
			Limit: rowsPerPage,
		},
	)
	if err != nil {
		http.Error(
			w,
			"failed to search companies",
			http.StatusInternalServerError,
		)
		return
	}

	options := make([]companyOption, len(rows))
	for i, row := range rows {
		options[i] = companyOption{
			ID:   row.ID,
			Name: row.Name,
		}
	}

	payload, err := json.Marshal(options)
	if err != nil {
		http.Error(
			w,
			"failed to encode companies",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

func (s *Server) handleCompany(w http.ResponseWriter, r *http.Request) {
	company, ok := s.loadCompany(w, r)
	if !ok {
		return
	}

	s.templates.Render(w, "company", PageData{
		Title:      company.Name,
		ActiveMenu: "companies",
		Data: CompanyPageData{
			Company: company,
			Saved:   r.URL.Query().Get("saved") == "1",
		},
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	rows := make([]campaigns.CampaignRowView, 0, len(s.campaigns))

	for _, c := range s.campaigns {
		rows = append(rows, campaigns.CampaignRowView{
			Code: c.Code(),
			Name: c.Name(),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Code < rows[j].Code
	})

	s.templates.Render(w, "config", PageData{
		Title:      "Configuration",
		ActiveMenu: "config",
		Data:       CampaignsPageData{Campaigns: rows},
	})
}

func (s *Server) handleCampaign(w http.ResponseWriter, r *http.Request) {
	campaign, exists := s.campaigns[r.PathValue("code")]
	if !exists {
		http.NotFound(w, r)
		return
	}

	personRows, err := s.enrollmentQueries.ListCampaignPersons(
		r.Context(),
		campaign.Code(),
	)
	if err != nil {
		http.Error(w, "failed to list campaign persons", http.StatusInternalServerError)
		return
	}

	persons := make([]CampaignPersonPageData, 0, len(personRows))
	for _, row := range personRows {
		persons = append(persons, CampaignPersonPageData{
			PersonID:   row.PersonID,
			PersonName: row.PersonName,
			CompanyID:  row.CompanyID,
			Company:    row.Company,
			Status:     enrollmentStateLabel(row.State),
		})
	}

	description, err := renderMarkdown(campaign.Description())
	if err != nil {
		description = "<p>Error: markdown engine failed to render campaign description</p>"
	}

	campaignSteps := campaign.Steps()
	steps := make([]CampaignStepPageData, 0, len(campaignSteps))
	for _, step := range campaignSteps {
		instructions, err := renderMarkdown(step.Instructions())
		if err != nil {
			instructions = "<p>Error: markdown engine failed to render step instructions</p>"
		}

		steps = append(steps, CampaignStepPageData{
			Step:         step,
			Instructions: instructions,
		})
	}

	s.templates.Render(w, "campaign", PageData{
		Title:      campaign.Name(),
		ActiveMenu: "config",
		Data: CampaignPageData{
			Campaign:    campaign,
			Description: description,
			Steps:       steps,
			Persons:     persons,
		},
	})
}

func (s *Server) loadCompany(
	w http.ResponseWriter,
	r *http.Request,
) (contacts.CompanyView, bool) {
	id, err := common.IDFromString(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return contacts.CompanyView{}, false
	}

	company, err := s.companyQueries.GetCompany(r.Context(), id)
	if errors.Is(err, contacts.ErrCompanyNotFound) {
		http.NotFound(w, r)
		return contacts.CompanyView{}, false
	}
	if err != nil {
		http.Error(w, "failed to get company", http.StatusInternalServerError)
		return contacts.CompanyView{}, false
	}

	return company, true
}

func (s *Server) handleNewCompany(w http.ResponseWriter, r *http.Request) {
	s.renderCompanyForm(w, CompanyFormData{
		Heading:     "New company",
		Action:      "/companies",
		SubmitLabel: "Create company",
	}, http.StatusOK)
}

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request) {
	input, err := parseCompanyInput(w, r)
	if err != nil {
		writeCompanyFormError(w, err)
		return
	}

	id, err := s.companyCommands.CreateCompany(r.Context(), input)
	if nameError := companyNameError(err); nameError != "" {
		s.renderCompanyForm(w, CompanyFormData{
			Heading:     "New company",
			Action:      "/companies",
			SubmitLabel: "Create company",
			Input:       input,
			NameError:   nameError,
		}, http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		http.Error(w, "failed to create company", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/companies/"+id.String()+"?saved=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleEditCompany(w http.ResponseWriter, r *http.Request) {
	company, ok := s.loadCompany(w, r)
	if !ok {
		return
	}

	s.renderCompanyForm(w, CompanyFormData{
		Heading:     "Edit company",
		Action:      "/companies/" + company.ID,
		SubmitLabel: "Save changes",
		Input: contacts.CompanyInput{
			Name:    company.Name,
			Country: company.Country,
		},
	}, http.StatusOK)
}

func (s *Server) handleUpdateCompany(w http.ResponseWriter, r *http.Request) {
	id, err := common.IDFromString(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	input, err := parseCompanyInput(w, r)
	if err != nil {
		writeCompanyFormError(w, err)
		return
	}

	err = s.companyCommands.UpdateCompany(r.Context(), id, input)
	if nameError := companyNameError(err); nameError != "" {
		s.renderCompanyForm(w, CompanyFormData{
			Heading:     "Edit company",
			Action:      "/companies/" + id.String(),
			SubmitLabel: "Save changes",
			Input:       input,
			NameError:   nameError,
		}, http.StatusUnprocessableEntity)
		return
	}
	if errors.Is(err, contacts.ErrCompanyNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to update company", http.StatusInternalServerError)
		return
	}

	http.Redirect(
		w,
		r,
		"/companies/"+id.String()+"?saved=1",
		http.StatusSeeOther,
	)
}

func (s *Server) handleDeleteCompany(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := common.IDFromString(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	err = s.companyCommands.DeleteCompany(r.Context(), id)
	if errors.Is(err, contacts.ErrCompanyNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "failed to delete company", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/companies", http.StatusSeeOther)
}

func (s *Server) renderPersonForm(w http.ResponseWriter, data PersonFormData, status int) {
	if status != http.StatusOK {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
	}
	s.templates.Render(w, "person_form", PageData{
		Title:      data.Heading,
		ActiveMenu: "persons",
		Data:       data,
	})
}

func parseSkip(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	skip, err := strconv.Atoi(value)
	if err != nil || skip < 0 {
		return 0, errors.New("invalid skip")
	}

	return skip, nil
}

func buildPersonsURL(query string, skip int) string {
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if skip > 0 {
		values.Set("skip", strconv.Itoa(skip))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/persons?" + encoded
	}
	return "/persons"
}

func buildCompaniesURL(query string, skip int) string {
	values := url.Values{}
	if query != "" {
		values.Set("q", query)
	}
	if skip > 0 {
		values.Set("skip", strconv.Itoa(skip))
	}
	if encoded := values.Encode(); encoded != "" {
		return "/companies?" + encoded
	}
	return "/companies"
}

func personNameError(err error) string {
	switch {
	case errors.Is(err, contacts.ErrPersonNameRequired):
		return "Name is required"
	case errors.Is(err, contacts.ErrPersonNameExists):
		return "A person with this name already exists"
	default:
		return ""
	}
}

func personCompanyError(err error) string {
	if errors.Is(err, contacts.ErrPersonCompanyNotFound) {
		return "Selected company no longer exists"
	}
	return ""
}

func parsePersonInput(
	w http.ResponseWriter,
	r *http.Request,
) (contacts.PersonInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPersonFormBodySize)
	var err error
	if err = r.ParseForm(); err != nil {
		return contacts.PersonInput{}, err
	}

	var companyID common.ID
	if value := r.PostForm.Get("company_id"); value != "" {
		companyID, err = common.IDFromString(value)
		if err != nil {
			return contacts.PersonInput{}, fmt.Errorf(
				"parse company id: %w",
				err,
			)
		}
	}

	return contacts.PersonInput{
		Name:      r.PostForm.Get("name"),
		Position:  r.PostForm.Get("position"),
		CompanyID: companyID,
		Note:      r.PostForm.Get("note"),
	}, nil
}

func writePersonFormError(w http.ResponseWriter, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		http.Error(
			w,
			"request body too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	http.Error(w, "invalid form", http.StatusBadRequest)
}

func writeEnrollmentFormError(w http.ResponseWriter, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		http.Error(
			w,
			"request body too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	http.Error(w, "invalid form", http.StatusBadRequest)
}

func parseEnrollmentForm(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxEnrollmentFormBodySize)
	return r.ParseForm()
}

func (s *Server) loadPerson(
	w http.ResponseWriter,
	r *http.Request,
) (contacts.PersonView, bool) {
	id, err := common.IDFromString(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return contacts.PersonView{}, false
	}

	person, err := s.personQueries.GetPerson(r.Context(), id)
	if errors.Is(err, contacts.ErrPersonNotFound) {
		http.NotFound(w, r)
		return contacts.PersonView{}, false
	}
	if err != nil {
		http.Error(w, "failed to get person", http.StatusInternalServerError)
		return contacts.PersonView{}, false
	}

	return person, true
}

func (s *Server) renderCompanyForm(
	w http.ResponseWriter,
	data CompanyFormData,
	status int,
) {
	if status != http.StatusOK {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(status)
	}

	s.templates.Render(w, "company_form", PageData{
		Title:      data.Heading,
		ActiveMenu: "companies",
		Data:       data,
	})
}

func parseCompanyInput(
	w http.ResponseWriter,
	r *http.Request,
) (contacts.CompanyInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCompanyFormBodySize)
	if err := r.ParseForm(); err != nil {
		return contacts.CompanyInput{}, err
	}

	return contacts.CompanyInput{
		Name:    r.PostForm.Get("name"),
		Country: r.PostForm.Get("country"),
	}, nil
}

func companyNameError(err error) string {
	switch {
	case errors.Is(err, contacts.ErrCompanyNameRequired):
		return "Name is required"
	case errors.Is(err, contacts.ErrCompanyNameExists):
		return "A company with this name already exists"
	default:
		return ""
	}
}

func writeCompanyFormError(w http.ResponseWriter, err error) {
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		http.Error(
			w,
			"request body too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	http.Error(w, "invalid form", http.StatusBadRequest)
}

func personCompanySelectionError(input contacts.PersonInput, companyName string) string {
	if input.CompanyID.IsZero() && strings.TrimSpace(companyName) != "" {
		return "Select a company from the list or clear the field"
	}
	return ""
}

func (s *Server) personEnrollmentPageData(
	rows []campaigns.PersonEnrollmentRowView,
) []PersonEnrollmentPageData {
	result := make([]PersonEnrollmentPageData, 0, len(rows))

	for _, row := range rows {
		campaignName := row.Campaign
		var moveSteps []campaigns.Step

		if campaign, exists := s.campaigns[row.Campaign]; exists {
			campaignName = campaign.Name()

			for _, step := range campaign.Steps() {
				if step.Code() != row.Step &&
					row.State == campaigns.EnrollmentActive {
					moveSteps = append(moveSteps, step)
				}
			}
		}

		result = append(result, PersonEnrollmentPageData{
			ID:           row.ID,
			CampaignCode: row.Campaign,
			CampaignName: campaignName,
			Status:       enrollmentStateLabel(row.State),
			Next:         row.Next.String(),
			Intention:    row.Intention,
			Active:       row.State == campaigns.EnrollmentActive,
			MoveSteps:    moveSteps,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CampaignName == result[j].CampaignName {
			return result[i].ID < result[j].ID
		}
		return result[i].CampaignName < result[j].CampaignName
	})

	return result
}

func (s *Server) availableCampaigns(
	rows []campaigns.PersonEnrollmentRowView,
) []campaigns.CampaignRowView {
	enrolled := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		enrolled[row.Campaign] = struct{}{}
	}

	result := make([]campaigns.CampaignRowView, 0, len(s.campaigns))
	for code, campaign := range s.campaigns {
		if _, exists := enrolled[code]; exists {
			continue
		}

		result = append(result, campaigns.CampaignRowView{
			Code: code,
			Name: campaign.Name(),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name == result[j].Name {
			return result[i].Code < result[j].Code
		}
		return result[i].Name < result[j].Name
	})

	return result
}

func enrollmentStateLabel(state campaigns.EnrollmentState) string {
	switch state {
	case campaigns.EnrollmentActive:
		return "Active"
	case campaigns.EnrollmentCompleted:
		return "Completed"
	case campaigns.EnrollmentStopped:
		return "Stopped"
	default:
		return string(state)
	}
}
