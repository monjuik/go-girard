package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/monjuik/go-girard/campaigns"
)

type Config struct {
	Campaigns map[string]campaigns.Campaign
}

type configFile struct {
	Campaigns []campaigns.CampaignConfig `json:"campaigns"`
}

func LoadConfig(path string) (Config, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return emptyConfig(), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	var source configFile

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&source); err != nil {
		return emptyConfig(), fmt.Errorf("decode config: %w", err)
	}

	// check if there are other root level elements
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, errors.New("decode config: multiple JSON values")
		}
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	config := emptyConfig()

	for i, c := range source.Campaigns {
		campaign, err := campaigns.NewCampaign(c)
		if err != nil {
			return Config{}, fmt.Errorf("campaigns[%d]: %w", i, err)
		}
		if _, exists := config.Campaigns[campaign.Code()]; exists {
			return Config{}, fmt.Errorf("campaigns[%d]: duplicate campaign code %q", i, campaign.Code())
		}
		config.Campaigns[campaign.Code()] = campaign
	}

	return config, nil
}

func emptyConfig() Config {
	return Config{
		Campaigns: make(map[string]campaigns.Campaign),
	}
}
