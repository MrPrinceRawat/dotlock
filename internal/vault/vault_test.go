package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".dotlock")

	salt := []byte("testsalt12345678")
	v := New(salt)
	v.SetEnv("default", "encrypted_data_here")
	v.SetEnv("prod", "encrypted_prod_data")

	if err := v.Save(path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.Version != CurrentVersion {
		t.Fatalf("version mismatch: %d", loaded.Version)
	}

	gotSalt, err := loaded.GetSalt()
	if err != nil {
		t.Fatal(err)
	}
	if string(gotSalt) != string(salt) {
		t.Fatal("salt mismatch")
	}

	ct, ok := loaded.GetEnv("default")
	if !ok || ct != "encrypted_data_here" {
		t.Fatalf("unexpected default env: %q", ct)
	}

	ct, ok = loaded.GetEnv("prod")
	if !ok || ct != "encrypted_prod_data" {
		t.Fatalf("unexpected prod env: %q", ct)
	}
}

func TestMultiEnv(t *testing.T) {
	v := New([]byte("salt1234567890ab"))
	v.SetEnv("default", "a")
	v.SetEnv("staging", "b")
	v.SetEnv("prod", "c")

	envs := v.ListEnvs()
	if len(envs) != 3 {
		t.Fatalf("expected 3 envs, got %d", len(envs))
	}
}

func TestCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".dotlock")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for corrupted file")
	}
}

func TestMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/.dotlock")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
