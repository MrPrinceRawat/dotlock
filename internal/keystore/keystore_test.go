package keystore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectHash(t *testing.T) {
	hash1, err := ProjectHash("/some/project")
	if err != nil {
		t.Fatal(err)
	}
	hash2, err := ProjectHash("/some/project")
	if err != nil {
		t.Fatal(err)
	}
	if hash1 != hash2 {
		t.Fatal("same path should produce same hash")
	}
	if len(hash1) != 64 {
		t.Fatalf("expected 64 char hex hash, got %d", len(hash1))
	}

	hash3, err := ProjectHash("/other/project")
	if err != nil {
		t.Fatal(err)
	}
	if hash1 == hash3 {
		t.Fatal("different paths should produce different hashes")
	}
}

func TestSaveLoadKey(t *testing.T) {
	// Use temp dir as home
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	projectPath := "/test/project/path"
	key := []byte("this-is-a-32-byte-encryption-key")

	if err := SaveKey(projectPath, key); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Verify directory permissions
	keysPath := filepath.Join(tmpHome, ".dotlock", "keys")
	info, err := os.Stat(keysPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("expected 0700 permissions, got %o", info.Mode().Perm())
	}

	loaded, err := LoadKey(projectPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if string(loaded) != string(key) {
		t.Fatal("loaded key doesn't match saved key")
	}
}

func TestLoadKeyMissing(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	loaded, err := LoadKey("/nonexistent/project")
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatal("expected nil for missing key")
	}
}

func TestResolveKeyFromEnvVar(t *testing.T) {
	os.Setenv("DOTLOCK_PASSPHRASE", "testpass")
	defer os.Unsetenv("DOTLOCK_PASSPHRASE")

	salt := []byte("testsalt12345678")
	deriveFn := func(pass string, s []byte) []byte {
		return []byte("derived-key-from-" + pass)
	}
	promptFn := func() (string, error) {
		t.Fatal("should not prompt when env var is set")
		return "", nil
	}

	key, source, err := ResolveKey("/any/path", salt, deriveFn, promptFn)
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceEnvVar {
		t.Fatalf("expected env source, got %s", source)
	}
	if string(key) != "derived-key-from-testpass" {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestResolveKeyFromCache(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)
	os.Unsetenv("DOTLOCK_PASSPHRASE")

	projectPath := "/test/cached/project"
	cachedKey := []byte("cached-key-32-bytes-long-enough!")
	SaveKey(projectPath, cachedKey)

	deriveFn := func(pass string, s []byte) []byte { return nil }
	promptFn := func() (string, error) {
		t.Fatal("should not prompt when cache exists")
		return "", nil
	}

	key, source, err := ResolveKey(projectPath, []byte("salt"), deriveFn, promptFn)
	if err != nil {
		t.Fatal(err)
	}
	if source != SourceCache {
		t.Fatalf("expected cache source, got %s", source)
	}
	if string(key) != string(cachedKey) {
		t.Fatal("key mismatch")
	}
}
