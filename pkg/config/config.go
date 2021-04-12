package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config is global config object
var Config *config

type config struct {
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
}

func init() {
	Config = newConfig()
}

func newConfig() *config {
	if Config != nil {
		return Config
	}
	newCfg := &config{}
	newCfg.FeatureGates = make(map[string]bool)
	return newCfg
}

// ReadConfig loads config
func (cfg *config) ReadConfig(configFile string) error {
	allCfg := make(map[string]json.RawMessage)
	rawBytes, err := ioutil.ReadFile(configFile)

	if err != nil {
		return fmt.Errorf("error reading file %s, %v", configFile, err)
	}

	if err = json.Unmarshal(rawBytes, &allCfg); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v", err)
	}

	fgMap := make(map[string]bool)

	if err = json.Unmarshal(allCfg["featureGates"], &fgMap); err != nil {
		return fmt.Errorf("error unmarshalling raw bytes %v", err)
	}

	for k, v := range fgMap {
		cfg.FeatureGates[k] = v
	}

	return nil
}
