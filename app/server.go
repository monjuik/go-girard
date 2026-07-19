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
	personsPerPage        = 20
	maxPersonFormBodySize = 64 << 10
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

type Server struct {
	personQueries  contacts.PersonQueries
	personCommands contacts.PersonCommands
	templates      *Templates
	httpServer     *http.Server
}

func NewServer(port int, personQueries contacts.PersonQueries, personCommands contacts.PersonCommands) (*Server, error) {
	templates, err := NewTemplates()
	if err != nil {
		return nil, err
	}
	server := &Server{
		personQueries:  personQueries,
		personCommands: personCommands,
		templates:      templates,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /persons", server.handlePersons)
	mux.HandleFunc("GET /persons/new", server.handleNewPerson)
	mux.HandleFunc("GET /persons/{id}", server.handlePerson)
	mux.HandleFunc("GET /persons/{id}/edit", server.handleEditPerson)
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
			Limit: personsPerPage + 1,
		},
	)
	if err != nil {
		http.Error(w, "failed to list persons", http.StatusInternalServerError)
		return
	}
	hasNext := len(persons) > personsPerPage
	if hasNext {
		persons = persons[:personsPerPage]
	}
	data := PersonsPageData{
		Persons: persons,
		Query:   query,
	}

	if skip > 0 {
		data.PreviousURL = buildPersonsURL(
			query,
			max(0, skip-personsPerPage),
		)
	}

	if hasNext {
		data.NextURL = buildPersonsURL(
			query,
			skip+personsPerPage,
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
