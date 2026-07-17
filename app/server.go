package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/monjuik/go-girard/contacts"
)

const personsPerPage = 20

type PersonsPageData struct {
	Persons     []contacts.PersonRowView
	Query       string
	PreviousURL string
	NextURL     string
}

type Server struct {
	app        *App
	templates  *Templates
	httpServer *http.Server
}

func NewServer(port int, personQueries contacts.PersonQueries) (*Server, error) {
	templates, err := NewTemplates()
	if err != nil {
		return nil, err
	}
	app := NewApp(personQueries)
	server := &Server{
		app:       app,
		templates: templates,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /persons", server.handlePersons)
	addr := fmt.Sprintf(":%d", port)
	server.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
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

	persons, err := s.app.ListPersonRows(
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
