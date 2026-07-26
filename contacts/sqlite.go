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

type SQLiteCompanyQueries struct {
	db *sql.DB
}

type SQLiteCompanyRepository struct {
	db *sql.DB
}

func NewSQLitePersonQueries(db *sql.DB) *SQLitePersonQueries {
	return &SQLitePersonQueries{db: db}
}

func NewSQLitePersonRepository(db *sql.DB) *SQLitePersonRepository {
	return &SQLitePersonRepository{db: db}
}

func NewSQLiteCompanyQueries(db *sql.DB) *SQLiteCompanyQueries {
	return &SQLiteCompanyQueries{db: db}
}

func NewSQLiteCompanyRepository(db *sql.DB) *SQLiteCompanyRepository {
	return &SQLiteCompanyRepository{db: db}
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
						AND company.deleted = 0
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
	var rawCompanyID int64

	err := q.db.QueryRowContext(
		ctx,
		`
			SELECT
				person.name,
				person.position,
				person.note,
				COALESCE(company.id, 0),
				COALESCE(company.name, '')
			FROM person
			LEFT JOIN company
				ON company.id = person.company
				AND company.deleted = 0
			WHERE person.id = ? AND person.deleted = 0
		`,
		id.Int64(),
	).Scan(&view.Name, &view.Position, &view.Note, &rawCompanyID, &view.CompanyName)
	if errors.Is(err, sql.ErrNoRows) {
		return PersonView{}, ErrPersonNotFound
	}
	if err != nil {
		return PersonView{}, fmt.Errorf("query person: %w", err)
	}

	companyID := common.ID(rawCompanyID)
	if companyID.IsValid() {
		view.CompanyID = companyID.String()
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
			INSERT INTO person (id, name, position, note, company)
			VALUES (?, ?, ?, ?, ?)
		`,
		person.ID().Int64(),
		person.Name(),
		person.Position(),
		person.Note(),
		personCompanyID(person),
	)
	if err != nil {
		switch {
		case isUniqueConstraint(err):
			return ErrPersonNameExists
		case isFKConstraint(err):
			return ErrPersonCompanyNotFound
		default:
			return fmt.Errorf("insert person: %w", err)
		}
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
			SET name = ?, position = ?, note = ?, company = ?
			WHERE id = ? AND deleted = 0
		`,
		person.Name(),
		person.Position(),
		person.Note(),
		personCompanyID(person),
		person.ID().Int64(),
	)
	if err != nil {
		switch {
		case isUniqueConstraint(err):
			return ErrPersonNameExists
		case isFKConstraint(err):
			return ErrPersonCompanyNotFound
		default:
			return fmt.Errorf("update person: %w", err)
		}
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

func (r *SQLitePersonRepository) Delete(ctx context.Context, id common.ID) error {
	result, err := r.db.ExecContext(
		ctx,
		`
			UPDATE person
			SET deleted = 1
			WHERE id = ? AND deleted = 0
		`,
		id.Int64(),
	)
	if err != nil {
		return fmt.Errorf("delete person: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted person count: %w", err)
	}
	if affected == 0 {
		return ErrPersonNotFound
	}
	return nil
}

func (q *SQLiteCompanyQueries) ListCompanyRows(
	ctx context.Context,
	filter CompaniesFilter,
) ([]CompanyRowView, error) {
	if filter.Skip < 0 {
		return nil, errors.New("skip cannot be negative")
	}
	if filter.Limit <= 0 {
		return nil, errors.New("limit must be positive")
	}

	search := strings.TrimSpace(filter.Query)
	pattern := "%" + escapeLike(search) + "%"

	rows, err := q.db.QueryContext(
		ctx,
		`
			SELECT id, name, country
			FROM company
			WHERE deleted = 0
			AND (
					? = ''
					OR name COLLATE NOCASE LIKE ? ESCAPE '\'
			)
			ORDER BY name COLLATE NOCASE, id
			LIMIT ? OFFSET ?
		`,
		search,
		pattern,
		filter.Limit,
		filter.Skip,
	)
	if err != nil {
		return nil, fmt.Errorf("query company rows: %w", err)
	}
	defer rows.Close()

	result := make([]CompanyRowView, 0, filter.Limit)
	for rows.Next() {
		var id int64
		var name, country string

		if err := rows.Scan(&id, &name, &country); err != nil {
			return nil, fmt.Errorf("scan company row: %w", err)
		}

		result = append(result, CompanyRowView{
			ID:      common.ID(id).String(),
			Name:    name,
			Country: country,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate company rows: %w", err)
	}

	return result, nil
}

func (q *SQLiteCompanyQueries) GetCompany(
	ctx context.Context,
	id common.ID,
) (CompanyView, error) {
	view := CompanyView{ID: id.String()}
	err := q.db.QueryRowContext(
		ctx,
		`
			SELECT name, country
			FROM company
			WHERE id = ? AND deleted = 0
		`,
		id.Int64(),
	).Scan(&view.Name, &view.Country)

	if errors.Is(err, sql.ErrNoRows) {
		return CompanyView{}, ErrCompanyNotFound
	}
	if err != nil {
		return CompanyView{}, fmt.Errorf("query company: %w", err)
	}

	return view, nil
}

func (r *SQLiteCompanyRepository) Add(
	ctx context.Context,
	company Company,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO company (id, name, country) VALUES (?, ?, ?)`,
		company.ID().Int64(),
		company.Name(),
		company.Country(),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrCompanyNameExists
		}
		return fmt.Errorf("insert company: %w", err)
	}

	return nil
}

func (r *SQLiteCompanyRepository) Save(
	ctx context.Context,
	company Company,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`
			UPDATE company
			SET name = ?, country = ?
			WHERE id = ? AND deleted = 0
		`,
		company.Name(),
		company.Country(),
		company.ID().Int64(),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrCompanyNameExists
		}
		return fmt.Errorf("update company: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated company count: %w", err)
	}
	if affected == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

func (r *SQLiteCompanyRepository) Delete(
	ctx context.Context,
	id common.ID,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`
			UPDATE company
			SET deleted = 1
			WHERE id = ? AND deleted = 0
		`,
		id.Int64(),
	)
	if err != nil {
		return fmt.Errorf("delete company: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get deleted company count: %w", err)
	}
	if affected == 0 {
		return ErrCompanyNotFound
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

func isFKConstraint(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintForeignKey
}

func personCompanyID(person Person) sql.NullInt64 {
	companyID := person.CompanyID()
	if companyID.IsZero() {
		return sql.NullInt64{}
	}

	return sql.NullInt64{
		Int64: companyID.Int64(),
		Valid: true,
	}
}
