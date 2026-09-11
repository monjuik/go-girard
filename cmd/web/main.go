package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/monjuik/go-girard/app"
	"github.com/monjuik/go-girard/campaigns"
	"github.com/monjuik/go-girard/contacts"
)

var version = "dev"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("go-girard", flag.ContinueOnError)
	port := flags.Int("port", 8080, "HTTP server port")
	dbPath := flags.String("db", "go-girard.db", "SQLite database path")
	configPath := flags.String("config", "config.json", "Config path")
	showVersion := flags.Bool("version", false, "Display version and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *showVersion {
		_, err := fmt.Fprintf(stdout, "go-girard %s\n", version)
		return err
	}

	config, err := app.LoadConfig(*configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx := context.Background()

	db, err := app.OpenDatabase(ctx, *dbPath)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	if err := app.Migrate(ctx, db); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	personQueries := contacts.NewSQLitePersonQueries(db)
	personRepository := contacts.NewSQLitePersonRepository(db)
	personCommands := contacts.NewPersonService(personRepository)

	companyQueries := contacts.NewSQLiteCompanyQueries(db)
	companyRepository := contacts.NewSQLiteCompanyRepository(db)
	companyCommands := contacts.NewCompanyService(companyRepository)

	enrollmentQueries := campaigns.NewSQLiteEnrollmentQueries(db)
	enrollmentRepository := campaigns.NewSQLiteEnrollmentRepository(db)
	enrollmentCommands := campaigns.NewEnrollmentService(enrollmentRepository, config.Campaigns)

	server, err := app.NewServer(
		*port,
		version,
		config.Campaigns,
		personQueries,
		personCommands,
		companyQueries,
		companyCommands,
		enrollmentQueries,
		enrollmentCommands,
	)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	slog.Info(
		"starting web server",
		"port", *port,
		"database", *dbPath,
		"campaigns", len(config.Campaigns),
	)

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
