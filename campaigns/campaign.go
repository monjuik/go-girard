package campaigns

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrCampaignCodeInvalid     = errors.New("campaign code is invalid")
	ErrCampaignNameRequired    = errors.New("campaign name is required")
	ErrCampaignVersionInvalid  = errors.New("campaign version is invalid")
	ErrCampaignStepsRequired   = errors.New("campaign steps are required")
	ErrStepCodeInvalid         = errors.New("step code is invalid")
	ErrStepCodeDuplicate       = errors.New("step code is duplicated")
	ErrStepNameRequired        = errors.New("step name is required")
	ErrEnrollmentPolicyInvalid = errors.New("enrollment policy is invalid")
)

var codePattern = regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)

type Campaign struct {
	code                string
	name                string
	version             int
	description         string
	maxActivePerCompany int
	steps               []Step
}

type Step struct {
	code         string
	name         string
	instructions string
}

type CampaignConfig struct {
	Code             string                  `json:"code"`
	Name             string                  `json:"name"`
	Version          int                     `json:"version"`
	Description      string                  `json:"description,omitempty"`
	EnrollmentPolicy *EnrollmentPolicyConfig `json:"enrollmentPolicy,omitempty"`
	Steps            []StepConfig            `json:"steps"`
}

type EnrollmentPolicyConfig struct {
	MaxActivePerCompany int `json:"maxActivePerCompany"`
}

type StepConfig struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Instructions string `json:"instructions,omitempty"`
}

type CampaignRowView struct {
	Code string
	Name string
}

func NewCampaign(config CampaignConfig) (Campaign, error) {
	if !codePattern.MatchString(config.Code) {
		return Campaign{}, ErrCampaignCodeInvalid
	}
	name := strings.TrimSpace(config.Name)

	if name == "" {
		return Campaign{}, ErrCampaignNameRequired
	}

	if config.Version < 1 {
		return Campaign{}, ErrCampaignVersionInvalid
	}

	if len(config.Steps) == 0 {
		return Campaign{}, ErrCampaignStepsRequired
	}

	maxActivePerCompany := 0
	if config.EnrollmentPolicy != nil {
		maxActivePerCompany = config.EnrollmentPolicy.MaxActivePerCompany
		if maxActivePerCompany < 1 {
			return Campaign{}, ErrEnrollmentPolicyInvalid
		}
	}

	steps := make([]Step, 0, len(config.Steps))
	stepCodes := make(map[string]struct{}, len(config.Steps))

	for i, c := range config.Steps {
		if !codePattern.MatchString(c.Code) {
			return Campaign{}, fmt.Errorf("step %d: %w", i, ErrStepCodeInvalid)
		}

		if _, exists := stepCodes[c.Code]; exists {
			return Campaign{}, fmt.Errorf("step %d %q: %w", i, c.Code, ErrStepCodeDuplicate)
		}

		name := strings.TrimSpace(c.Name)
		if name == "" {
			return Campaign{}, fmt.Errorf("step %d: %w", i, ErrStepNameRequired)
		}
		stepCodes[c.Code] = struct{}{}
		steps = append(steps, Step{
			code:         c.Code,
			name:         name,
			instructions: c.Instructions,
		})
	}

	return Campaign{
		code:                config.Code,
		name:                name,
		version:             config.Version,
		description:         config.Description,
		maxActivePerCompany: maxActivePerCompany,
		steps:               steps,
	}, nil
}

func (c Campaign) Code() string {
	return c.code
}

func (c Campaign) Name() string {
	return c.name
}

func (c Campaign) Version() int {
	return c.version
}

func (c Campaign) Description() string {
	return c.description
}

func (c Campaign) MaxActivePerCompany() int {
	return c.maxActivePerCompany
}

func (c Campaign) Steps() []Step {
	return append([]Step(nil), c.steps...)
}

func (s Step) Code() string {
	return s.code
}

func (s Step) Name() string {
	return s.name
}

func (s Step) Instructions() string {
	return s.instructions
}
