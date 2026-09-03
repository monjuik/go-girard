package campaigns

import (
	"context"

	"github.com/monjuik/go-girard/common"
)

type EnrollmentQueries interface {
	ListPersonEnrollments(ctx context.Context, personID common.ID) ([]PersonEnrollmentRowView, error)

	ListCampaignPersons(ctx context.Context, campaign string) ([]CampaignPersonRowView, error)

	ListDueEnrollments(ctx context.Context, through common.Date) ([]DueEnrollmentRowView, error)
}
