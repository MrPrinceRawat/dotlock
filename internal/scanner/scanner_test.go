package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPatternMatching(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"stripe live key", `key := "sk_live_1234567890abcdefgh1234"`, "Stripe Secret Key"},
		{"aws key", `AWS_KEY=AKIAIOSFODNN7EXAMPLE`, "AWS Access Key"},
		{"github token", `token := "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij"`, "GitHub Token"},
		{"postgres conn", `db := "postgres://user:password@localhost/db"`, "Postgres Connection"},
		{"redis conn", `r := "redis://admin:secret@redis.example.com:6379"`, "Redis Connection"},
		{"hardcoded password", `password = "supersecretpassword"`, "Hardcoded Password"},
		{"private key", `-----BEGIN RSA PRIVATE KEY-----`, "Private Key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "test.go")
			os.WriteFile(path, []byte(tt.content), 0644)

			findings, err := ScanFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(findings) == 0 {
				t.Fatalf("expected finding for %s", tt.name)
			}
			if findings[0].Type != tt.want {
				t.Fatalf("expected type %q, got %q", tt.want, findings[0].Type)
			}
		})
	}
}

func TestNoFalsePositives(t *testing.T) {
	dir := t.TempDir()
	content := `package main

func main() {
	fmt.Println("hello world")
	x := 42
}
`
	path := filepath.Join(dir, "main.go")
	os.WriteFile(path, []byte(content), 0644)

	findings, err := ScanFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %d", len(findings))
	}
}

func TestScanDirSkipsDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create node_modules with a secret
	nmDir := filepath.Join(dir, "node_modules", "pkg")
	os.MkdirAll(nmDir, 0755)
	os.WriteFile(filepath.Join(nmDir, "index.js"), []byte(`key = "sk_live_1234567890abcdefgh1234"`), 0644)

	// Create a normal file with a secret
	os.WriteFile(filepath.Join(dir, "config.js"), []byte(`key = "sk_live_1234567890abcdefgh1234"`), 0644)

	findings, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Should only find the one in config.js, not node_modules
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].File != filepath.Join(dir, "config.js") {
		t.Fatalf("unexpected file: %s", findings[0].File)
	}
}

func TestScanDirSkipsEnvFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".env"), []byte(`STRIPE_KEY=sk_live_1234567890abcdefgh1234`), 0644)
	os.WriteFile(filepath.Join(dir, "app.py"), []byte(`x = "hello"`), 0644)

	findings, err := ScanDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings (env files should be skipped), got %d", len(findings))
	}
}
