package app

import (
	"fmt"
	"net/http"

	"github.com/monjuik/go-girard/contacts"
)

type Server struct {
	app        *App
	templates  *Templates
	httpServer *http.Server
}

func NewServer(port int) (*Server, error) {
	templates, err := NewTemplates()
	if err != nil {
		return nil, err
	}
	app := NewApp()
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
	persons, err := s.app.ListPersonRows(r.Context(), contacts.PersonsFilter{})
	if err != nil {
		http.Error(w, "failed to list persons", http.StatusInternalServerError)
		return
	}
	s.templates.Render(w, "persons", PageData{
		Title:      "Persons",
		ActiveMenu: "persons",
		Data:       persons,
	})
}
