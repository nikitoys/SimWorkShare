package runprofile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Profile struct {
	BaseConfig        string `json:"base_config"`
	Scenario          string `json:"scenario"`
	BehaviorCase      string `json:"behavior_case"`
	EnvironmentCase   string `json:"environment_case"`
	CalibrationStatus string `json:"calibration_status"`
}

func LoadFile(path string) (Profile, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, "", fmt.Errorf("read run profile %q: %w", path, err)
	}
	if err := validateNoDuplicateKeys(data); err != nil {
		return Profile{}, "", fmt.Errorf("decode run profile %q: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var profile Profile
	if err := decoder.Decode(&profile); err != nil {
		return Profile{}, "", fmt.Errorf("decode run profile %q: %w", path, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Profile{}, "", fmt.Errorf("decode run profile %q: multiple JSON values", path)
		}
		return Profile{}, "", fmt.Errorf("decode run profile %q: %w", path, err)
	}

	for _, item := range []struct {
		name  string
		value string
	}{
		{"base_config", profile.BaseConfig},
		{"scenario", profile.Scenario},
		{"behavior_case", profile.BehaviorCase},
	} {
		if strings.TrimSpace(item.value) == "" {
			return Profile{}, "", fmt.Errorf("run profile field %s must not be empty", item.name)
		}
	}
	if profile.EnvironmentCase != "normal_market" {
		return Profile{}, "", fmt.Errorf("run profile environment_case must be %q in this stage", "normal_market")
	}
	if profile.CalibrationStatus != "template_defaults_not_real_data" && profile.CalibrationStatus != "calibrated" {
		return Profile{}, "", fmt.Errorf("unsupported calibration_status %q", profile.CalibrationStatus)
	}

	baseConfigPath := profile.BaseConfig
	if !filepath.IsAbs(baseConfigPath) {
		baseConfigPath = filepath.Join(filepath.Dir(path), baseConfigPath)
	}
	baseConfigPath, err = filepath.Abs(baseConfigPath)
	if err != nil {
		return Profile{}, "", fmt.Errorf("resolve base_config %q: %w", profile.BaseConfig, err)
	}
	return profile, filepath.Clean(baseConfigPath), nil
}

func validateNoDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	opening, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := opening.(json.Delim); !ok || delimiter != '{' {
		return fmt.Errorf("run profile must be a JSON object")
	}
	seen := make(map[string]bool)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("run profile field name must be a string")
		}
		if seen[key] {
			return fmt.Errorf("%s: duplicate field", key)
		}
		seen[key] = true
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	return nil
}
