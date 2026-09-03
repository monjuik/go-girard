package campaigns

import (
	"context"

	"github.com/monjuik/go-girard/common"
)

type EnrollmentCommands interface {
	// personID as a paramenter is here on purpose.
	// these commands are accessible from the person page, so we need to check the ownership

	Enroll(ctx context.Context, personID common.ID, campaignCode string) (common.ID, error)
	Postpone(ctx context.Context, personID common.ID, enrollmentID common.ID, next common.Date) error
	Move(
		ctx context.Context,
		personID common.ID,
		enrollmentID common.ID,
		step string,
		next common.Date,
		intention string,
	) error
	Complete(ctx context.Context, personID common.ID, enrollmentID common.ID) error
	Stop(ctx context.Context, personID common.ID, enrollmentID common.ID) error
}
