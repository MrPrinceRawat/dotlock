package vault

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

const CurrentVersion = 1

// Vault represents the .dotlock file structure.
type Vault struct {
	Version      int               `json:"version"`
	Salt         string            `json:"salt"`
	Environments map[string]string `json:"environments"`
}

// New creates a new vault with the given salt.
func New(salt []byte) *Vault {
	return &Vault{
		Version:      CurrentVersion,
		Salt:         base64.StdEncoding.EncodeToString(salt),
		Environments: make(map[string]string),
	}
}

// Load reads a vault from a .dotlock file.
func Load(path string) (*Vault, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading vault: %w", err)
	}

	var v Vault
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("parsing vault: %w", err)
	}
	if v.Environments == nil {
		v.Environments = make(map[string]string)
	}
	return &v, nil
}

// Save writes the vault to a .dotlock file.
func (v *Vault) Save(path string) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling vault: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

// GetSalt returns the decoded salt bytes.
func (v *Vault) GetSalt() ([]byte, error) {
	return base64.StdEncoding.DecodeString(v.Salt)
}

// SetEnv stores an encrypted environment blob.
func (v *Vault) SetEnv(name, ciphertext string) {
	v.Environments[name] = ciphertext
}

// GetEnv returns the encrypted blob for an environment.
func (v *Vault) GetEnv(name string) (string, bool) {
	ct, ok := v.Environments[name]
	return ct, ok
}

// ListEnvs returns all environment names.
func (v *Vault) ListEnvs() []string {
	names := make([]string, 0, len(v.Environments))
	for name := range v.Environments {
		names = append(names, name)
	}
	return names
}
