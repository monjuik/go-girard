package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"
	"github.com/monjuik/go-girard/common"
)

type SQLiteEnrollmentRepository struct {
	db *sql.DB
}

type SQLiteEnrollmentQueries struct {
	db *sql.DB
}

var _ EnrollmentRepository = (*SQLiteEnrollmentRepository)(nil)
var _ EnrollmentQueries = (*SQLiteEnrollmentQueries)(nil)

func NewSQLiteEnrollmentRepository(db *sql.DB) *SQLiteEnrollmentRepository {
	return &SQLiteEnrollmentRepository{db: db}
}

func NewSQLiteEnrollmentQueries(db *sql.DB) *SQLiteEnrollmentQueries {
	return &SQLiteEnrollmentQueries{db: db}
}

func (r *SQLiteEnrollmentRepository) Get(
	ctx context.Context,
	id common.ID,
) (Enrollment, error) {
	var (
		campaign  string
		step      string
		rawPerson int64
		rawState  string
		rawNext   sql.NullString
		intention string
	)

	err := r.db.QueryRowContext(
		ctx,
		`
  			SELECT
  				enrollment.campaign,
  				enrollment.step,
  				enrollment.person,
  				enrollment.state,
  				enrollment.next,
  				enrollment.intention
  			FROM enrollment
  			JOIN person
  				ON person.id = enrollment.person
  				AND person.deleted = 0
  			WHERE enrollment.id = ?
  		`,
		id.Int64(),
	).Scan(
		&campaign,
		&step,
		&rawPerson,
		&rawState,
		&rawNext,
		&intention,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Enrollment{}, ErrEnrollmentNotFound
	}
	if err != nil {
		return Enrollment{}, fmt.Errorf("query enrollment: %w", err)
	}

	next, err := parseEnrollmentNext(rawNext)
	if err != nil {
		return Enrollment{}, err
	}

	enrollment, err := NewEnrollment(
		id,
		campaign,
		step,
		common.ID(rawPerson),
		EnrollmentState(rawState),
		next,
		intention,
	)
	if err != nil {
		return Enrollment{}, fmt.Errorf(
			"build enrollment from db: %w",
			err,
		)
	}

	return enrollment, nil
}

func (r *SQLiteEnrollmentRepository) Save(
	ctx context.Context,
	enrollment Enrollment,
) error {
	result, err := r.db.ExecContext(
		ctx,
		`
			UPDATE enrollment
			SET
				step = ?,
				state = ?,
				next = ?,
				intention = ?
			WHERE id = ?
				AND person = ?
				AND person IN (
					SELECT id
					FROM person
					WHERE deleted = 0
				)
		`,
		enrollment.Step(),
		string(enrollment.State()),
		enrollmentNextValue(enrollment.Next()),
		enrollment.Intention(),
		enrollment.ID().Int64(),
		enrollment.PersonID().Int64(),
	)
	if err != nil {
		return fmt.Errorf("update enrollment: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"get updated enrollment count: %w",
			err,
		)
	}
	if affected == 0 {
		return ErrEnrollmentNotFound
	}

	return nil
}

func (r *SQLiteEnrollmentRepository) Add(
	ctx context.Context,
	enrollment Enrollment,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`
			INSERT INTO enrollment (
				id,
				campaign,
				step,
				person,
				state,
				next,
				intention
			)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
		enrollment.ID().Int64(),
		enrollment.Campaign(),
		enrollment.Step(),
		enrollment.PersonID().Int64(),
		string(enrollment.State()),
		enrollmentNextValue(enrollment.Next()),
		enrollment.Intention(),
	)
	if err != nil {
		return enrollmentInsertError(err)
	}

	return nil
}

func (q *SQLiteEnrollmentQueries) ListPersonEnrollments(
	ctx context.Context,
	personID common.ID,
) ([]PersonEnrollmentRowView, error) {
	if !personID.IsValid() {
		return nil, ErrEnrollmentPersonIDInvalid
	}

	rows, err := q.db.QueryContext(
		ctx,
		`
			SELECT
				enrollment.id,
				enrollment.campaign,
				enrollment.step,
				enrollment.state,
				enrollment.next,
				enrollment.intention
			FROM enrollment
			JOIN person
				ON person.id = enrollment.person
				AND person.deleted = 0
			WHERE enrollment.person = ?
			ORDER BY enrollment.campaign, enrollment.id
		`,
		personID.Int64(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query person enrollments: %w",
			err,
		)
	}
	defer rows.Close()

	result := make([]PersonEnrollmentRowView, 0)
	for rows.Next() {
		var (
			rawID    int64
			rawState string
			rawNext  sql.NullString
			row      PersonEnrollmentRowView
		)

		if err := rows.Scan(
			&rawID,
			&row.Campaign,
			&row.Step,
			&rawState,
			&rawNext,
			&row.Intention,
		); err != nil {
			return nil, fmt.Errorf(
				"scan person enrollment: %w",
				err,
			)
		}

		next, err := parseEnrollmentNext(rawNext)
		if err != nil {
			return nil, err
		}

		row.ID = common.ID(rawID).String()
		row.State = EnrollmentState(rawState)
		row.Next = next
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate person enrollments: %w",
			err,
		)
	}

	return result, nil
}

func (q *SQLiteEnrollmentQueries) ListCampaignPersons(
	ctx context.Context,
	campaign string,
) ([]CampaignPersonRowView, error) {
	if !codePattern.MatchString(campaign) {
		return nil, ErrEnrollmentCampaignInvalid
	}

	rows, err := q.db.QueryContext(
		ctx,
		`
			SELECT
				person.id,
				person.name,
				COALESCE(company.id, 0),
				COALESCE(company.name, ''),
				enrollment.state
			FROM enrollment
			JOIN person
				ON person.id = enrollment.person
				AND person.deleted = 0
			LEFT JOIN company
				ON company.id = person.company
				AND company.deleted = 0
			WHERE enrollment.campaign = ?
			ORDER BY person.name COLLATE NOCASE, person.id
		`,
		campaign,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query campaign persons: %w",
			err,
		)
	}
	defer rows.Close()

	result := make([]CampaignPersonRowView, 0)
	for rows.Next() {
		var (
			rawPersonID  int64
			rawCompanyID int64
			rawState     string
			row          CampaignPersonRowView
		)

		if err := rows.Scan(
			&rawPersonID,
			&row.PersonName,
			&rawCompanyID,
			&row.Company,
			&rawState,
		); err != nil {
			return nil, fmt.Errorf(
				"scan campaign person: %w",
				err,
			)
		}

		row.PersonID = common.ID(rawPersonID).String()
		row.State = EnrollmentState(rawState)

		companyID := common.ID(rawCompanyID)
		if companyID.IsValid() {
			row.CompanyID = companyID.String()
		}

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate campaign persons: %w",
			err,
		)
	}

	return result, nil
}

func (q *SQLiteEnrollmentQueries) ListDueEnrollments(
	ctx context.Context,
	through common.Date,
) ([]DueEnrollmentRowView, error) {
	if through.IsZero() {
		return nil, ErrEnrollmentNextInvalid
	}

	rows, err := q.db.QueryContext(
		ctx,
		`
			SELECT
				enrollment.id,
				person.id,
				person.name,
				enrollment.campaign,
				enrollment.step,
				enrollment.next,
				enrollment.intention
			FROM enrollment
			JOIN person
				ON person.id = enrollment.person
				AND person.deleted = 0
			WHERE enrollment.state = ?
				AND enrollment.next <= ?
			ORDER BY
				enrollment.next,
				person.name COLLATE NOCASE,
				enrollment.id
		`,
		string(EnrollmentActive),
		through.String(),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query due enrollments: %w",
			err,
		)
	}
	defer rows.Close()

	result := make([]DueEnrollmentRowView, 0)
	for rows.Next() {
		var (
			rawID       int64
			rawPersonID int64
			rawNext     string
			row         DueEnrollmentRowView
		)

		if err := rows.Scan(
			&rawID,
			&rawPersonID,
			&row.PersonName,
			&row.Campaign,
			&row.Step,
			&rawNext,
			&row.Intention,
		); err != nil {
			return nil, fmt.Errorf(
				"scan due enrollment: %w",
				err,
			)
		}

		next, err := common.ParseDate(rawNext)
		if err != nil {
			return nil, fmt.Errorf(
				"parse due enrollment date: %w",
				err,
			)
		}

		row.ID = common.ID(rawID).String()
		row.PersonID = common.ID(rawPersonID).String()
		row.Next = next
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate due enrollments: %w",
			err,
		)
	}

	return result, nil
}

func enrollmentNextValue(next common.Date) any {
	if next.IsZero() {
		return nil
	}
	return next.String()
}

func enrollmentInsertError(err error) error {
	if sqliteErr, ok := errors.AsType[sqlite3.Error](err); ok {
		switch sqliteErr.ExtendedCode {
		case sqlite3.ErrConstraintUnique:
			return ErrEnrollmentExists
		case sqlite3.ErrConstraintForeignKey:
			return ErrEnrollmentPersonNotFound
		}
	}

	return fmt.Errorf("insert enrollment: %w", err)
}

func parseEnrollmentNext(
	rawNext sql.NullString,
) (common.Date, error) {
	if !rawNext.Valid {
		return common.Date{}, nil
	}

	next, err := common.ParseDate(rawNext.String)
	if err != nil {
		return common.Date{}, fmt.Errorf(
			"parse enrollment next date: %w",
			err,
		)
	}
	return next, nil
}
