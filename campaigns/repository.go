package campaigns

import (
	"context"
	"errors"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrEnrollmentNotFound       = errors.New("enrollment not found")
	ErrEnrollmentExists         = errors.New("enrollment already exists")
	ErrEnrollmentPersonNotFound = errors.New("enrollment person not found")
)

type EnrollmentRepository interface {
	Add(ctx context.Context, enrollment Enrollment) error

	Get(ctx context.Context, id common.ID) (Enrollment, error)

	Save(ctx context.Context, enrollment Enrollment) error
}
