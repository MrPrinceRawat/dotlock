package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mrprincerawat/dotlock/internal/crypto"
	"github.com/mrprincerawat/dotlock/internal/envfile"
	"github.com/mrprincerawat/dotlock/internal/keystore"
	"github.com/mrprincerawat/dotlock/internal/vault"
)

func TestEndToEnd(t *testing.T) {
	// Setup temp directory with .env files
	dir := t.TempDir()

	envContent := "DB_HOST=localhost\nDB_PORT=5432\nDB_PASS=supersecret\n"
	envProdContent := "DB_HOST=prod.example.com\nDB_PORT=5432\nDB_PASS=prodpassword\n"

	os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644)
	os.WriteFile(filepath.Join(dir, ".env.prod"), []byte(envProdContent), 0644)

	passphrase := "test-passphrase-123"
	vaultPath := filepath.Join(dir, ".dotlock")

	// === INIT: encrypt .env files ===
	salt, err := crypto.GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}

	key := crypto.DeriveKey(passphrase, salt)
	v := vault.New(salt)

	envFiles, err := envfile.DetectEnvFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(envFiles) != 2 {
		t.Fatalf("expected 2 env files, got %d", len(envFiles))
	}

	for _, envPath := range envFiles {
		content, err := os.ReadFile(envPath)
		if err != nil {
			t.Fatal(err)
		}
		encrypted, err := crypto.Encrypt(content, key)
		if err != nil {
			t.Fatal(err)
		}
		name := envfile.EnvName(envPath)
		v.SetEnv(name, encrypted)
	}

	if err := v.Save(vaultPath); err != nil {
		t.Fatal(err)
	}

	// Save key to cache
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	if err := keystore.SaveKey(dir, key); err != nil {
		t.Fatal(err)
	}

	// === DELETE: remove .env files ===
	os.Remove(filepath.Join(dir, ".env"))
	os.Remove(filepath.Join(dir, ".env.prod"))

	// Verify they're gone
	if _, err := os.Stat(filepath.Join(dir, ".env")); !os.IsNotExist(err) {
		t.Fatal(".env should be deleted")
	}

	// === UNLOCK: decrypt from vault ===
	v2, err := vault.Load(vaultPath)
	if err != nil {
		t.Fatal(err)
	}

	salt2, err := v2.GetSalt()
	if err != nil {
		t.Fatal(err)
	}

	key2 := crypto.DeriveKey(passphrase, salt2)

	for _, name := range v2.ListEnvs() {
		ct, _ := v2.GetEnv(name)
		plaintext, err := crypto.Decrypt(ct, key2)
		if err != nil {
			t.Fatalf("decrypt %s: %v", name, err)
		}

		fileName := envfile.EnvFileName(name)
		outPath := filepath.Join(dir, fileName)
		if err := os.WriteFile(outPath, plaintext, 0644); err != nil {
			t.Fatal(err)
		}
	}

	// === VERIFY: contents match original ===
	restored, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != envContent {
		t.Fatalf("default env mismatch:\ngot:  %q\nwant: %q", restored, envContent)
	}

	restoredProd, err := os.ReadFile(filepath.Join(dir, ".env.prod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(restoredProd) != envProdContent {
		t.Fatalf("prod env mismatch:\ngot:  %q\nwant: %q", restoredProd, envProdContent)
	}

	// === VERIFY: wrong passphrase fails ===
	wrongKey := crypto.DeriveKey("wrong-passphrase", salt2)
	ct, _ := v2.GetEnv("default")
	_, err = crypto.Decrypt(ct, wrongKey)
	if err == nil {
		t.Fatal("expected error for wrong passphrase")
	}
}
