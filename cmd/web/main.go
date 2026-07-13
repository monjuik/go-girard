package main

import (
	"flag"
	"log/slog"
	"os"

	app "github.com/monjuik/go-girard/app"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	server, err := app.NewServer(*port)
	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("starting web server", "port", *port)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("web server stopped", "error", err)
		os.Exit(1)
	}
}
