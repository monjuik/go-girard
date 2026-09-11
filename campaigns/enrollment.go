package campaigns

import (
	"errors"
	"strings"

	"github.com/monjuik/go-girard/common"
)

var (
	ErrEnrollmentIDInvalid         = errors.New("enrollment id is invalid")
	ErrEnrollmentCampaignInvalid   = errors.New("enrollment campaign is invalid")
	ErrEnrollmentStepInvalid       = errors.New("enrollment step is invalid")
	ErrEnrollmentStepUnchanged     = errors.New("enrollment step is unchanged")
	ErrEnrollmentPersonIDInvalid   = errors.New("enrollment person id is invalid")
	ErrEnrollmentStateInvalid      = errors.New("enrollment state is invalid")
	ErrEnrollmentNextInvalid       = errors.New("enrollment next date is invalid")
	ErrEnrollmentDateNotFuture     = errors.New("enrollment date must be in the future")
	ErrEnrollmentIntentionRequired = errors.New("enrollment intention is required")
	ErrEnrollmentInactive          = errors.New("enrollment is not active")
)

type EnrollmentState string

const (
	EnrollmentActive    EnrollmentState = "active"
	EnrollmentCompleted EnrollmentState = "completed"
	EnrollmentStopped   EnrollmentState = "stopped"
)

type Enrollment struct {
	id        common.ID
	campaign  string
	step      string
	personID  common.ID
	state     EnrollmentState
	next      common.Date
	intention string
}

// PersonEnrollmentRowView contains enrollment data displayed
// on an individual person page.
type PersonEnrollmentRowView struct {
	ID        string
	Campaign  string
	Step      string
	State     EnrollmentState
	Next      common.Date
	Intention string
}

// CampaignPersonRowView contains participant data displayed
// on an individual campaign page.
type CampaignPersonRowView struct {
	PersonID   string
	PersonName string
	CompanyID  string
	Company    string
	State      EnrollmentState
}

// DueEnrollmentRowView contains an enrollment displayed
// in the Dashboard due-action queue.
type DueEnrollmentRowView struct {
	ID         string
	PersonID   string
	PersonName string
	Campaign   string
	Step       string
	Next       common.Date
	Intention  string
}

func NewEnrollment(
	id common.ID,
	campaign string,
	step string,
	personID common.ID,
	state EnrollmentState,
	next common.Date,
	intention string,
) (Enrollment, error) {
	if !id.IsValid() {
		return Enrollment{}, ErrEnrollmentIDInvalid
	}
	if !codePattern.MatchString(campaign) {
		return Enrollment{}, ErrEnrollmentCampaignInvalid
	}
	if !codePattern.MatchString(step) {
		return Enrollment{}, ErrEnrollmentStepInvalid
	}
	if !personID.IsValid() {
		return Enrollment{}, ErrEnrollmentPersonIDInvalid
	}
	if !validEnrollmentState(state) {
		return Enrollment{}, ErrEnrollmentStateInvalid
	}

	switch state {
	case EnrollmentActive:
		if next.IsZero() {
			return Enrollment{}, ErrEnrollmentNextInvalid
		}
	case EnrollmentCompleted, EnrollmentStopped:
		if !next.IsZero() {
			return Enrollment{}, ErrEnrollmentNextInvalid
		}
	}

	intention = strings.TrimSpace(intention)
	if intention == "" {
		return Enrollment{}, ErrEnrollmentIntentionRequired
	}

	return Enrollment{
		id:        id,
		campaign:  campaign,
		step:      step,
		personID:  personID,
		state:     state,
		next:      next,
		intention: intention,
	}, nil
}

func (e *Enrollment) Postpone(next, today common.Date) error {
	if e.state != EnrollmentActive {
		return ErrEnrollmentInactive
	}
	if err := validateFutureEnrollmentDate(next, today); err != nil {
		return err
	}

	e.next = next
	return nil
}

func (e *Enrollment) Move(
	step string,
	next common.Date,
	intention string,
	today common.Date,
) error {
	if e.state != EnrollmentActive {
		return ErrEnrollmentInactive
	}
	if !codePattern.MatchString(step) {
		return ErrEnrollmentStepInvalid
	}
	if step == e.step {
		return ErrEnrollmentStepUnchanged
	}
	if err := validateFutureEnrollmentDate(next, today); err != nil {
		return err
	}

	intention = strings.TrimSpace(intention)
	if intention == "" {
		return ErrEnrollmentIntentionRequired
	}

	e.step = step
	e.next = next
	e.intention = intention
	return nil
}

func (e *Enrollment) Stop() error {
	if e.state != EnrollmentActive {
		return ErrEnrollmentInactive
	}

	e.state = EnrollmentStopped
	e.next = common.Date{}
	return nil
}

func (e *Enrollment) Complete() error {
	if e.state != EnrollmentActive {
		return ErrEnrollmentInactive
	}

	e.state = EnrollmentCompleted
	e.next = common.Date{}
	return nil
}

func (e Enrollment) ID() common.ID {
	return e.id
}

func (e Enrollment) Campaign() string {
	return e.campaign
}

func (e Enrollment) Step() string {
	return e.step
}

func (e Enrollment) PersonID() common.ID {
	return e.personID
}

func (e Enrollment) State() EnrollmentState {
	return e.state
}

func (e Enrollment) Next() common.Date {
	return e.next
}

func (e Enrollment) Intention() string {
	return e.intention
}

func validEnrollmentState(state EnrollmentState) bool {
	switch state {
	case EnrollmentActive, EnrollmentCompleted, EnrollmentStopped:
		return true
	default:
		return false
	}
}

func validateFutureEnrollmentDate(next, today common.Date) error {
	if next.IsZero() || today.IsZero() {
		return ErrEnrollmentNextInvalid
	}
	if !next.IsAfter(today) {
		return ErrEnrollmentDateNotFuture
	}
	return nil
}
