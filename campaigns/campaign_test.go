package campaigns

import (
	"errors"
	"testing"
)

func TestNewCampaign(t *testing.T) {
	config := validCampaignConfig()

	campaign, err := NewCampaign(config)
	if err != nil {
		t.Fatalf("NewCampaign() error = %v", err)
	}

	if campaign.Code() != "linkedInOutreach" {
		t.Fatalf(
			"campaign.Code() = %q, want %q",
			campaign.Code(),
			"linkedInOutreach",
		)
	}
	if campaign.Name() != "LinkedIn outreach" {
		t.Fatalf(
			"campaign.Name() = %q, want %q",
			campaign.Name(),
			"LinkedIn outreach",
		)
	}
	if campaign.Version() != 2 {
		t.Fatalf("campaign.Version() = %d, want 2", campaign.Version())
	}
	if campaign.Description() != "Campaign description" {
		t.Fatalf(
			"campaign.Description() = %q, want %q",
			campaign.Description(),
			"Campaign description",
		)
	}
	steps := campaign.Steps()
	if len(steps) != 2 {
		t.Fatalf("len(campaign.Steps()) = %d, want 2", len(steps))
	}

	if steps[0].Code() != "discoverPlans" ||
		steps[0].Name() != "Discover plans" ||
		steps[0].Instructions() != "Ask open questions." {
		t.Fatalf("campaign.Steps()[0] = %#v, want first configured step", steps[0])
	}

	if steps[1].Code() != "agreeNextAction" {
		t.Fatalf(
			"campaign.Steps()[1].Code() = %q, want %q",
			steps[1].Code(),
			"agreeNextAction",
		)
	}

	steps[0] = Step{}
	if campaign.Steps()[0].Code() != "discoverPlans" {
		t.Fatal("modifying Steps() result changed campaign")
	}
}

func TestNewCampaignValidation(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*CampaignConfig)
		wantErr error
	}{
		{
			name: "invalid campaign code",
			change: func(config *CampaignConfig) {
				config.Code = "LinkedIn-Outreach"
			},
			wantErr: ErrCampaignCodeInvalid,
		},
		{
			name: "blank campaign name",
			change: func(config *CampaignConfig) {
				config.Name = " \t "
			},
			wantErr: ErrCampaignNameRequired,
		},
		{
			name: "invalid version",
			change: func(config *CampaignConfig) {
				config.Version = 0
			},
			wantErr: ErrCampaignVersionInvalid,
		},
		{
			name: "no steps",
			change: func(config *CampaignConfig) {
				config.Steps = nil
			},
			wantErr: ErrCampaignStepsRequired,
		},
		{
			name: "invalid step code",
			change: func(config *CampaignConfig) {
				config.Steps[0].Code = "discover_plans"
			},
			wantErr: ErrStepCodeInvalid,
		},
		{
			name: "duplicate step code",
			change: func(config *CampaignConfig) {
				config.Steps[1].Code = config.Steps[0].Code
			},
			wantErr: ErrStepCodeDuplicate,
		},
		{
			name: "blank step name",
			change: func(config *CampaignConfig) {
				config.Steps[0].Name = " \t "
			},
			wantErr: ErrStepNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := validCampaignConfig()
			tt.change(&config)

			_, err := NewCampaign(config)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"NewCampaign() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func validCampaignConfig() CampaignConfig {
	return CampaignConfig{
		Code:        "linkedInOutreach",
		Name:        "  LinkedIn outreach  ",
		Version:     2,
		Description: "Campaign description",
		Steps: []StepConfig{
			{
				Code:         "discoverPlans",
				Name:         "  Discover plans  ",
				Instructions: "Ask open questions.",
			},
			{
				Code:         "agreeNextAction",
				Name:         "Agree next action",
				Instructions: "Set a follow-up date.",
			},
		},
	}
}
