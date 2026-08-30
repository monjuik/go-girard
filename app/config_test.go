package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		config, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}
		if config.Campaigns == nil || len(config.Campaigns) != 0 {
			t.Fatalf("Campaigns = %#v, want empty map", config.Campaigns)
		}
	})

	t.Run("valid file", func(t *testing.T) {
		path := writeTestConfig(t, `{
  			"campaigns": [{
  				"code": "linkedInOutreach",
  				"name": "LinkedIn outreach",
  				"version": 1,
  				"steps": [{
  					"code": "discoverPlans",
  					"name": "Discover plans"
  				}]
  			}]
  		}`)

		config, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		campaign, exists := config.Campaigns["linkedInOutreach"]
		if !exists {
			t.Fatal(`Campaigns["linkedInOutreach"] does not exist`)
		}
		if campaign.Name() != "LinkedIn outreach" {
			t.Fatalf(
				"campaign.Name() = %q, want %q",
				campaign.Name(),
				"LinkedIn outreach",
			)
		}
	})
}

func TestLoadConfigErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "invalid JSON",
			content: `{`,
			want:    "decode config",
		},
		{
			name:    "unknown field",
			content: `{"unexpected": true}`,
			want:    `unknown field "unexpected"`,
		},
		{
			name: "invalid campaign",
			content: `{
  				"campaigns": [{
  					"code": "Invalid-Code"
  				}]
  			}`,
			want: "campaigns[0]: campaign code is invalid",
		},
		{
			name: "duplicate campaign code",
			content: `{
  				"campaigns": [
  					{
  						"code": "followUp",
  						"name": "Follow-up",
  						"version": 1,
  						"steps": [{"code": "contact", "name": "Contact"}]
  					},
  					{
  						"code": "followUp",
  						"name": "Another follow-up",
  						"version": 2,
  						"steps": [{"code": "contact", "name": "Contact"}]
  					}
  				]
  			}`,
			want: `duplicate campaign code "followUp"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(writeTestConfig(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadConfig() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
