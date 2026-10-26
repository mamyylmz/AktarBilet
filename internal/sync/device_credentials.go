package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadDeviceCredentials reads per-device secrets from a runtime-only JSON file.
// The file must contain an object whose keys are validator IDs and whose values
// are unique secrets of at least 32 bytes. A shared/master fallback is
// intentionally unsupported.
func LoadDeviceCredentials(path string) (map[string]string, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("VALIDATOR_CREDENTIALS_FILE is required")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("validator credentials file cannot be read: %w", err)
	}

	credentials := make(map[string]string)
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return nil, fmt.Errorf("validator credentials file is not valid JSON: %w", err)
	}
	if len(credentials) == 0 {
		return nil, fmt.Errorf("validator credentials file is empty")
	}

	seen := make(map[string]string, len(credentials))
	for validatorID, secret := range credentials {
		if strings.TrimSpace(validatorID) == "" {
			return nil, fmt.Errorf("validator credentials contain an empty validator ID")
		}
		if len(secret) < 32 {
			return nil, fmt.Errorf("validator credential for %s must be at least 32 bytes", validatorID)
		}
		if previousID, ok := seen[secret]; ok {
			return nil, fmt.Errorf("validator credentials for %s and %s are not unique", previousID, validatorID)
		}
		seen[secret] = validatorID
	}

	return credentials, nil
}
