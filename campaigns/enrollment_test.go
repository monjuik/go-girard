package campaigns

import (
	"errors"
	"testing"

	"github.com/monjuik/go-girard/common"
)

func TestNewEnrollment(t *testing.T) {
	next := mustEnrollmentDate(t, "2026-08-31")

	enrollment, err := NewEnrollment(
		common.ID(201),
		"followUp",
		"restoreContext",
		common.ID(101),
		EnrollmentActive,
		next,
		"  Restore context  ",
	)
	if err != nil {
		t.Fatalf("NewEnrollment() error = %v", err)
	}

	if enrollment.ID() != common.ID(201) {
		t.Fatalf("ID() = %d, want 201", enrollment.ID())
	}
	if enrollment.Campaign() != "followUp" {
		t.Fatalf(
			"Campaign() = %q, want %q",
			enrollment.Campaign(),
			"followUp",
		)
	}
	if enrollment.Step() != "restoreContext" {
		t.Fatalf(
			"Step() = %q, want %q",
			enrollment.Step(),
			"restoreContext",
		)
	}
	if enrollment.PersonID() != common.ID(101) {
		t.Fatalf("PersonID() = %d, want 101", enrollment.PersonID())
	}
	if enrollment.State() != EnrollmentActive {
		t.Fatalf(
			"State() = %q, want %q",
			enrollment.State(),
			EnrollmentActive,
		)
	}
	if enrollment.Next() != next {
		t.Fatalf(
			"Next() = %q, want %q",
			enrollment.Next(),
			next,
		)
	}
	if enrollment.Intention() != "Restore context" {
		t.Fatalf(
			"Intention() = %q, want %q",
			enrollment.Intention(),
			"Restore context",
		)
	}
}

func TestNewEnrollmentDateInvariant(t *testing.T) {
	next := mustEnrollmentDate(t, "2026-08-31")

	_, err := NewEnrollment(
		common.ID(201),
		"followUp",
		"restoreContext",
		common.ID(101),
		EnrollmentActive,
		common.Date{},
		"Restore context",
	)
	if !errors.Is(err, ErrEnrollmentNextInvalid) {
		t.Fatalf(
			"active enrollment error = %v, want ErrEnrollmentNextInvalid",
			err,
		)
	}

	_, err = NewEnrollment(
		common.ID(201),
		"followUp",
		"restoreContext",
		common.ID(101),
		EnrollmentStopped,
		next,
		"Restore context",
	)
	if !errors.Is(err, ErrEnrollmentNextInvalid) {
		t.Fatalf(
			"stopped enrollment error = %v, want ErrEnrollmentNextInvalid",
			err,
		)
	}
}

func TestEnrollmentPostpone(t *testing.T) {
	today := mustEnrollmentDate(t, "2026-08-31")
	tomorrow := mustEnrollmentDate(t, "2026-09-01")
	enrollment := newTestEnrollment(t, today)

	err := enrollment.Postpone(today, today)
	if !errors.Is(err, ErrEnrollmentDateNotFuture) {
		t.Fatalf(
			"Postpone() error = %v, want ErrEnrollmentDateNotFuture",
			err,
		)
	}
	if enrollment.Next() != today {
		t.Fatal("failed Postpone() changed enrollment")
	}

	if err := enrollment.Postpone(tomorrow, today); err != nil {
		t.Fatalf("Postpone() error = %v", err)
	}
	if enrollment.Next() != tomorrow {
		t.Fatalf(
			"Next() = %q, want %q",
			enrollment.Next(),
			tomorrow,
		)
	}
}

func TestEnrollmentMoveAndStop(t *testing.T) {
	today := mustEnrollmentDate(t, "2026-08-31")
	next := mustEnrollmentDate(t, "2026-09-01")
	enrollment := newTestEnrollment(t, today)

	err := enrollment.Move(
		"restoreContext",
		next,
		"Discover changes",
		today,
	)
	if !errors.Is(err, ErrEnrollmentStepUnchanged) {
		t.Fatalf(
			"Move() error = %v, want ErrEnrollmentStepUnchanged",
			err,
		)
	}

	if err := enrollment.Move(
		"discoverChanges",
		next,
		"  Discover changes  ",
		today,
	); err != nil {
		t.Fatalf("Move() error = %v", err)
	}

	if enrollment.Step() != "discoverChanges" {
		t.Fatalf(
			"Step() = %q, want %q",
			enrollment.Step(),
			"discoverChanges",
		)
	}
	if enrollment.Next() != next {
		t.Fatalf("Next() = %q, want %q", enrollment.Next(), next)
	}
	if enrollment.Intention() != "Discover changes" {
		t.Fatalf(
			"Intention() = %q, want %q",
			enrollment.Intention(),
			"Discover changes",
		)
	}

	if err := enrollment.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if enrollment.State() != EnrollmentStopped {
		t.Fatalf(
			"State() = %q, want %q",
			enrollment.State(),
			EnrollmentStopped,
		)
	}
	if !enrollment.Next().IsZero() {
		t.Fatalf(
			"Next() = %q after Stop(), want zero date",
			enrollment.Next(),
		)
	}

	if err := enrollment.Postpone(next, today); !errors.Is(
		err,
		ErrEnrollmentInactive,
	) {
		t.Fatalf(
			"Postpone() after Stop() error = %v, want ErrEnrollmentInactive",
			err,
		)
	}
}

func TestEnrollmentComplete(t *testing.T) {
	next := mustEnrollmentDate(t, "2026-09-01")
	enrollment := newTestEnrollment(t, next)

	if err := enrollment.Complete(); err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if enrollment.State() != EnrollmentCompleted {
		t.Fatalf(
			"State() = %q, want %q",
			enrollment.State(),
			EnrollmentCompleted,
		)
	}
	if !enrollment.Next().IsZero() {
		t.Fatalf(
			"Next() = %q after Complete(), want zero date",
			enrollment.Next(),
		)
	}
	if enrollment.Step() != "restoreContext" {
		t.Fatalf(
			"Step() = %q after Complete(), want restoreContext",
			enrollment.Step(),
		)
	}
	if enrollment.Intention() != "Restore context" {
		t.Fatalf(
			"Intention() = %q after Complete(), want Restore context",
			enrollment.Intention(),
		)
	}

	if err := enrollment.Complete(); !errors.Is(err, ErrEnrollmentInactive) {
		t.Fatalf(
			"second Complete() error = %v, want ErrEnrollmentInactive",
			err,
		)
	}
}

func newTestEnrollment(
	t *testing.T,
	next common.Date,
) Enrollment {
	t.Helper()

	enrollment, err := NewEnrollment(
		common.ID(201),
		"followUp",
		"restoreContext",
		common.ID(101),
		EnrollmentActive,
		next,
		"Restore context",
	)
	if err != nil {
		t.Fatalf("NewEnrollment() error = %v", err)
	}
	return enrollment
}

func mustEnrollmentDate(t *testing.T, value string) common.Date {
	t.Helper()

	date, err := common.ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) error = %v", value, err)
	}
	return date
}
