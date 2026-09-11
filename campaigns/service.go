package campaigns

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/monjuik/go-girard/common"
)

var ErrCampaignNotFound = errors.New("campaign not found")

type EnrollmentService struct {
	enrollments EnrollmentRepository
	campaigns   map[string]Campaign
	today       func() common.Date
}

// comptime check public contract
var _ EnrollmentCommands = (*EnrollmentService)(nil)

func NewEnrollmentService(
	enrollments EnrollmentRepository,
	campaigns map[string]Campaign,
) *EnrollmentService {
	return &EnrollmentService{
		enrollments: enrollments,
		campaigns:   campaigns,
		today: func() common.Date {
			return common.DateFromTime(time.Now())
		},
	}
}

func (s *EnrollmentService) Enroll(ctx context.Context, personID common.ID, campaignCode string) (common.ID, error) {
	if !personID.IsValid() {
		return 0, ErrEnrollmentPersonIDInvalid
	}

	campaign, exists := s.campaigns[campaignCode]
	if !exists {
		return 0, ErrCampaignNotFound
	}

	steps := campaign.Steps()
	firstStep := steps[0]
	enrollment, err := NewEnrollment(
		common.NewID(),
		campaign.Code(),
		firstStep.Code(),
		personID,
		EnrollmentActive,
		s.today(),
		firstStep.Name(),
	)
	if err != nil {
		return 0, err
	}
	if err := s.enrollments.Add(ctx, enrollment); err != nil {
		return 0, fmt.Errorf("add enrollment: %w", err)
	}

	return enrollment.ID(), nil
}

func (s *EnrollmentService) Postpone(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
	next common.Date,
) error {
	enrollment, err := s.loadEnrollment(
		ctx,
		personID,
		enrollmentID,
	)
	if err != nil {
		return err
	}

	if err := enrollment.Postpone(next, s.today()); err != nil {
		return err
	}

	if err := s.enrollments.Save(ctx, enrollment); err != nil {
		return fmt.Errorf("save enrollment: %w", err)
	}
	return nil
}

func (s *EnrollmentService) Move(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
	step string,
	next common.Date,
	intention string,
) error {
	enrollment, err := s.loadEnrollment(
		ctx,
		personID,
		enrollmentID,
	)
	if err != nil {
		return err
	}

	campaign, exists := s.campaigns[enrollment.Campaign()]
	if !exists {
		return ErrCampaignNotFound
	}
	if _, exists := campaign.FindStep(step); !exists {
		return ErrEnrollmentStepInvalid
	}

	if err := enrollment.Move(
		step,
		next,
		intention,
		s.today(),
	); err != nil {
		return err
	}

	if err := s.enrollments.Save(ctx, enrollment); err != nil {
		return fmt.Errorf("save enrollment: %w", err)
	}
	return nil
}

func (s *EnrollmentService) Stop(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
) error {
	enrollment, err := s.loadEnrollment(
		ctx,
		personID,
		enrollmentID,
	)
	if err != nil {
		return err
	}

	if err := enrollment.Stop(); err != nil {
		return err
	}

	if err := s.enrollments.Save(ctx, enrollment); err != nil {
		return fmt.Errorf("save enrollment: %w", err)
	}
	return nil
}

func (s *EnrollmentService) Complete(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
) error {
	enrollment, err := s.loadEnrollment(
		ctx,
		personID,
		enrollmentID,
	)
	if err != nil {
		return err
	}

	if err := enrollment.Complete(); err != nil {
		return err
	}

	if err := s.enrollments.Save(ctx, enrollment); err != nil {
		return fmt.Errorf("save enrollment: %w", err)
	}
	return nil
}

func (s *EnrollmentService) loadEnrollment(
	ctx context.Context,
	personID common.ID,
	enrollmentID common.ID,
) (Enrollment, error) {
	if !personID.IsValid() || !enrollmentID.IsValid() {
		return Enrollment{}, ErrEnrollmentNotFound
	}

	enrollment, err := s.enrollments.Get(ctx, enrollmentID)
	if err != nil {
		return Enrollment{}, fmt.Errorf("get enrollment: %w", err)
	}
	if enrollment.PersonID() != personID {
		return Enrollment{}, ErrEnrollmentNotFound
	}

	return enrollment, nil
}
