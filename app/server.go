package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/monjuik/go-girard/common"
	"github.com/monjuik/go-girard/contacts"
)

const (
	rowsPerPage            = 20
	maxPersonFormBodySize  = 64 << 10
	maxCompanyFormBodySize = 64 << 10
)

type PersonsPageData struct {
	Persons     []contacts.PersonRowView
	Query       string
	PreviousURL string
	NextURL     string
}

// PersonPageData contains data for the read-only person page.
type PersonPageData struct {
	Person contacts.PersonView
	Saved  bool
}

// PersonFormData contains data for the create and edit form.
type PersonFormData struct {
	Heading     string
	Action      string
	SubmitLabel string
	Input       contacts.PersonInput
	NameError   string
}

type CompaniesPageData struct {
	Companies   []contacts.CompanyRowView
	Query       string
	PreviousURL string
	NextURL     string
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

type Server struct {
	personQueries   contacts.PersonQueries
	personCommands  contacts.PersonCommands
	companyQueries  contacts.CompanyQueries
	companyCommands contacts.CompanyCommands
	templates       *Templates
	httpServer      *http.Server
}

func NewServer(
	port int,
	personQueries contacts.PersonQueries,
	personCommands contacts.PersonCommands,
	companyQueries contacts.CompanyQueries,
	companyCommands contacts.CompanyCommands,
) (*Server, error) {
	templates, err := NewTemplates()
	if err != nil {
		return nil, err
	}
	server := &Server{
		personQueries:   personQueries,
		personCommands:  personCommands,
		companyQueries:  companyQueries,
		companyCommands: companyCommands,
		templates:       templates,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /persons", server.handlePersons)
	mux.HandleFunc("GET /persons/new", server.handleNewPerson)
	mux.HandleFunc("GET /persons/{id}", server.handlePerson)
	mux.HandleFunc("GET /persons/{id}/edit", server.handleEditPerson)
	mux.HandleFunc("GET /companies", server.handleCompanies)
	mux.HandleFunc("GET /companies/new", server.handleNewCompany)
	mux.HandleFunc("GET /companies/{id}", server.handleCompany)
	mux.HandleFunc("GET /companies/{id}/edit", server.handleEditCompany)
	mux.HandleFunc("POST /companies", server.handleCreateCompany)
	mux.HandleFunc("POST /companies/{id}", server.handleUpdateCompany)
	mux.HandleFunc("POST /companies/{id}/delete", server.handleDeleteCompany)
	mux.HandleFunc("POST /persons", server.handleCreatePerson)
	mux.HandleFunc("POST /persons/{id}", server.handleUpdatePerson)
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

	s.templates.Render(w, "person", PageData{
		Title:      person.Name,
		ActiveMenu: "persons",
		Data: PersonPageData{
			Person: person,
			Saved:  r.URL.Query().Get("saved") == "1",
		},
	})
}

func (s *Server) handleEditPerson(w http.ResponseWriter, r *http.Request) {
	person, ok := s.loadPerson(w, r)
	if !ok {
		return
	}

	s.renderPersonForm(w, PersonFormData{
		Heading:     "Edit person",
		Action:      "/persons/" + person.ID,
		SubmitLabel: "Save changes",
		Input: contacts.PersonInput{
			Name:     person.Name,
			Position: person.Position,
		},
	}, http.StatusOK)
}

func (s *Server) handleCreatePerson(w http.ResponseWriter, r *http.Request) {
	input, err := parsePersonInput(w, r)
	if err != nil {
		writePersonFormError(w, err)
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

	err = s.personCommands.UpdatePerson(r.Context(), id, input)
	if nameError := personNameError(err); nameError != "" {
		s.renderPersonForm(w, PersonFormData{
			Heading:     "Edit person",
			Action:      "/persons/" + id.String(),
			SubmitLabel: "Save changes",
			Input:       input,
			NameError:   nameError,
		}, http.StatusUnprocessableEntity)
		return
	}
	if errors.Is(err, contacts.ErrPersonNotFound) {
		http.NotFound(w, r)
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

func (s *Server) handleCompanies(w http.ResponseWriter, r *http.Request) {
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

func parsePersonInput(
	w http.ResponseWriter,
	r *http.Request,
) (contacts.PersonInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPersonFormBodySize)
	if err := r.ParseForm(); err != nil {
		return contacts.PersonInput{}, err
	}

	return contacts.PersonInput{
		Name:     r.PostForm.Get("name"),
		Position: r.PostForm.Get("position"),
	}, nil
}

func writePersonFormError(w http.ResponseWriter, err error) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		http.Error(
			w,
			"request body too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	http.Error(w, "invalid form", http.StatusBadRequest)
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
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		http.Error(
			w,
			"request body too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	http.Error(w, "invalid form", http.StatusBadRequest)
}
