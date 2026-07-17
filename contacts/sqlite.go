package contacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/monjuik/go-girard/common"
)

type SQLitePersonQueries struct {
	db *sql.DB
}

func NewSQLitePersonQueries(db *sql.DB) *SQLitePersonQueries {
	return &SQLitePersonQueries{db: db}
}

func (q *SQLitePersonQueries) ListPersonRows(
	ctx context.Context,
	filter PersonsFilter,
) ([]PersonRowView, error) {
	if filter.Skip < 0 {
		return nil, errors.New("skip cannot be negative")
	}
	if filter.Limit <= 0 {
		return nil, errors.New("limit must be positive")
	}
	search := strings.TrimSpace(filter.Query)
	pattern := "%" + escapeLike(search) + "%"

	// Wildcards make regular SQLite indexes unusable for this
	rows, err := q.db.QueryContext(
		ctx,
		`
			SELECT
						person.id,
						person.name,
						person.position,
						COALESCE(company.name, '')
		 			FROM person
		 			LEFT JOIN company ON company.id = person.company
		 			WHERE
						? = ''
						OR person.name COLLATE NOCASE LIKE ? ESCAPE '\'
						OR person.position COLLATE NOCASE LIKE ? ESCAPE '\'
						OR company.name COLLATE NOCASE LIKE ? ESCAPE '\'
		 			ORDER BY person.name COLLATE NOCASE, person.id
		 			LIMIT ? OFFSET ?
		`,
		search,
		pattern,
		pattern,
		pattern,
		filter.Limit,
		filter.Skip,
	)
	if err != nil {
		return nil, fmt.Errorf("query person rows: %w", err)
	}
	defer rows.Close()

	result := make([]PersonRowView, 0, filter.Limit)
	for rows.Next() {
		var id int64
		var name, position, company string

		if err := rows.Scan(&id, &name, &position, &company); err != nil {
			return nil, fmt.Errorf("scan person row: %w", err)
		}
		result = append(result, PersonRowView{
			ID:       common.ID(id).String(),
			Name:     name,
			Position: position,
			Company:  company,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate person rows: %w", err)
	}

	return result, nil
}

var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

func escapeLike(value string) string {
	return likeEscaper.Replace(value)
}
