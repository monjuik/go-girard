package contacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"
	"github.com/monjuik/go-girard/common"
)

type SQLitePersonQueries struct {
	db *sql.DB
}

type SQLitePersonRepository struct {
	db *sql.DB
}

func NewSQLitePersonQueries(db *sql.DB) *SQLitePersonQueries {
	return &SQLitePersonQueries{db: db}
}

func NewSQLitePersonRepository(db *sql.DB) *SQLitePersonRepository {
	return &SQLitePersonRepository{db: db}
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
					WHERE person.deleted = 0
					AND (
						? = ''
						OR person.name COLLATE NOCASE LIKE ? ESCAPE '\'
						OR person.position COLLATE NOCASE LIKE ? ESCAPE '\'
						OR company.name COLLATE NOCASE LIKE ? ESCAPE '\'
					)
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

func (q *SQLitePersonQueries) GetPerson(
	ctx context.Context,
	id common.ID,
) (PersonView, error) {
	view := PersonView{ID: id.String()}
	err := q.db.QueryRowContext(
		ctx,
		`
			SELECT name, position
			FROM person
			WHERE id = ? AND deleted = 0
		`,
		id.Int64(),
	).Scan(&view.Name, &view.Position)
	if errors.Is(err, sql.ErrNoRows) {
		return PersonView{}, ErrPersonNotFound
	}
	if err != nil {
		return PersonView{}, fmt.Errorf("query person: %w", err)
	}
	return view, nil
}

func (r *SQLitePersonRepository) Add(
	ctx context.Context,
	person Person,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`
			INSERT INTO person (id, name, position, company)
			VALUES (?, ?, ?, ?)
		`,
		person.ID().Int64(),
		person.Name(),
		person.Position(),
		personCompanyID(person),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrPersonNameExists
		}
		return fmt.Errorf("insert person: %w", err)
	}

	return nil
}

func (r *SQLitePersonRepository) Save(
	ctx context.Context,
	person Person,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`
			UPDATE person
			SET name = ?, position = ?
			WHERE id = ? AND deleted = 0
		`,
		person.Name(),
		person.Position(),
		person.ID().Int64(),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrPersonNameExists
		}
		return fmt.Errorf("update person: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated person count: %w", err)
	}
	if affected == 0 {
		return ErrPersonNotFound
	}

	return nil
}

var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

func escapeLike(value string) string {
	return likeEscaper.Replace(value)
}

func isUniqueConstraint(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}

func personCompanyID(person Person) sql.NullInt64 {
	company := person.Company()
	if company == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{
		Int64: company.ID().Int64(),
		Valid: true,
	}
}
