package keystore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const (
	keysDirName = "keys"
	dirPerms    = 0700
	filePerms   = 0600
)

// KeySource indicates where the key was resolved from.
type KeySource string

const (
	SourceEnvVar KeySource = "env"
	SourceCache  KeySource = "cache"
	SourcePrompt KeySource = "prompt"
)

// dotlockDir returns the path to ~/.dotlock/.
func dotlockDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".dotlock"), nil
}

// keysDir returns the path to ~/.dotlock/keys/.
func keysDir() (string, error) {
	base, err := dotlockDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, keysDirName), nil
}

// EnsureKeysDir creates ~/.dotlock/keys/ if it doesn't exist.
func EnsureKeysDir() error {
	dir, err := keysDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, dirPerms)
}

// ProjectHash returns the SHA-256 hex hash of the absolute project path.
func ProjectHash(projectPath string) (string, error) {
	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return "", fmt.Errorf("getting absolute path: %w", err)
	}
	h := sha256.Sum256([]byte(abs))
	return hex.EncodeToString(h[:]), nil
}

// SaveKey saves a derived key to ~/.dotlock/keys/{hash}.key.
func SaveKey(projectPath string, key []byte) error {
	if err := EnsureKeysDir(); err != nil {
		return err
	}

	hash, err := ProjectHash(projectPath)
	if err != nil {
		return err
	}

	dir, err := keysDir()
	if err != nil {
		return err
	}

	keyPath := filepath.Join(dir, hash+".key")
	return os.WriteFile(keyPath, key, filePerms)
}

// LoadKey loads a cached key for the given project path.
// Returns nil, nil if no cached key exists.
func LoadKey(projectPath string) ([]byte, error) {
	hash, err := ProjectHash(projectPath)
	if err != nil {
		return nil, err
	}

	dir, err := keysDir()
	if err != nil {
		return nil, err
	}

	keyPath := filepath.Join(dir, hash+".key")
	data, err := os.ReadFile(keyPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading cached key: %w", err)
	}
	return data, nil
}

// ResolveKey resolves the encryption key from multiple sources in priority order:
// 1. DOTLOCK_PASSPHRASE env var
// 2. Cached key in ~/.dotlock/keys/
// 3. Interactive prompt (handled by caller via promptFn)
//
// promptFn should prompt the user for a passphrase and return (passphrase, error).
// deriveKeyFn derives a key from passphrase and salt.
func ResolveKey(projectPath string, salt []byte, deriveKeyFn func(string, []byte) []byte, promptFn func() (string, error)) ([]byte, KeySource, error) {
	// 1. Check env var
	if passphrase := os.Getenv("DOTLOCK_PASSPHRASE"); passphrase != "" {
		key := deriveKeyFn(passphrase, salt)
		return key, SourceEnvVar, nil
	}

	// 2. Check cache
	cached, err := LoadKey(projectPath)
	if err != nil {
		return nil, "", fmt.Errorf("checking key cache: %w", err)
	}
	if cached != nil {
		return cached, SourceCache, nil
	}

	// 3. Prompt
	passphrase, err := promptFn()
	if err != nil {
		return nil, "", fmt.Errorf("prompting for passphrase: %w", err)
	}

	key := deriveKeyFn(passphrase, salt)

	// Cache the derived key
	if err := SaveKey(projectPath, key); err != nil {
		// Non-fatal: warn but continue
		fmt.Fprintf(os.Stderr, "warning: could not cache key: %v\n", err)
	}

	return key, SourcePrompt, nil
}
