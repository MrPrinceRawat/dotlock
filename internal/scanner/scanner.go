package scanner

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Finding represents a detected secret in the codebase.
type Finding struct {
	File    string
	Line    int
	Type    string
	Content string
}

// Pattern defines a secret detection pattern.
type Pattern struct {
	Name  string
	Regex *regexp.Regexp
}

var DefaultPatterns = []Pattern{
	{Name: "Stripe Secret Key", Regex: regexp.MustCompile(`sk_live_[a-zA-Z0-9]{20,}`)},
	{Name: "Stripe Test Key", Regex: regexp.MustCompile(`sk_test_[a-zA-Z0-9]{20,}`)},
	{Name: "AWS Access Key", Regex: regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{Name: "GitHub Token", Regex: regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`)},
	{Name: "GitHub OAuth", Regex: regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`)},
	{Name: "GitLab Token", Regex: regexp.MustCompile(`glpat-[a-zA-Z0-9\-]{20,}`)},
	{Name: "Slack Token", Regex: regexp.MustCompile(`xox[bpors]-[a-zA-Z0-9\-]+`)},
	{Name: "Postgres Connection", Regex: regexp.MustCompile(`postgres://[^\s"']+:[^\s"']+@`)},
	{Name: "MySQL Connection", Regex: regexp.MustCompile(`mysql://[^\s"']+:[^\s"']+@`)},
	{Name: "MongoDB Connection", Regex: regexp.MustCompile(`mongodb(\+srv)?://[^\s"']+:[^\s"']+@`)},
	{Name: "Redis Connection", Regex: regexp.MustCompile(`redis://[^\s"']+:[^\s"']+@`)},
	{Name: "Hardcoded Password", Regex: regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["'][^"']{8,}["']`)},
	{Name: "Hardcoded Secret", Regex: regexp.MustCompile(`(?i)(secret|api_key|apikey|access_token)\s*[:=]\s*["'][^"']{8,}["']`)},
	{Name: "Private Key", Regex: regexp.MustCompile(`-----BEGIN (RSA |EC |DSA )?PRIVATE KEY-----`)},
}

// skipDirs lists directories to always skip.
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
	".venv":        true,
}

// binaryExtensions lists file extensions to skip.
var binaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".zip": true, ".tar": true, ".gz": true, ".bz2": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true, ".svg": true,
	".pdf": true, ".woff": true, ".woff2": true, ".ttf": true, ".eot": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true,
	".db": true, ".sqlite": true,
}

// ScanDir walks a directory and scans files for secrets.
func ScanDir(dir string) ([]Finding, error) {
	var findings []Finding

	// Try to read .gitignore for extra ignore patterns
	gitignorePatterns := loadGitignore(dir)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		if info.IsDir() {
			base := filepath.Base(path)
			if skipDirs[base] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip binary files
		ext := strings.ToLower(filepath.Ext(path))
		if binaryExtensions[ext] {
			return nil
		}

		// Skip .env files (they're expected to have secrets)
		base := filepath.Base(path)
		if strings.HasPrefix(base, ".env") {
			return nil
		}

		// Skip .dotlock file
		if base == ".dotlock" {
			return nil
		}

		// Skip gitignored files (basic matching)
		rel, _ := filepath.Rel(dir, path)
		for _, p := range gitignorePatterns {
			if matched, _ := filepath.Match(p, rel); matched {
				return nil
			}
			if matched, _ := filepath.Match(p, base); matched {
				return nil
			}
		}

		// Skip large files (> 1MB)
		if info.Size() > 1024*1024 {
			return nil
		}

		fileFindings, err := ScanFile(path)
		if err != nil {
			return nil // skip files we can't read
		}
		findings = append(findings, fileFindings...)
		return nil
	})

	return findings, err
}

// ScanFile scans a single file for secret patterns.
func ScanFile(path string) ([]Finding, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, p := range DefaultPatterns {
			if p.Regex.MatchString(line) {
				findings = append(findings, Finding{
					File:    path,
					Line:    lineNum,
					Type:    p.Name,
					Content: truncate(strings.TrimSpace(line), 120),
				})
				break // one finding per line
			}
		}
	}
	return findings, scanner.Err()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func loadGitignore(dir string) []string {
	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return nil
	}

	var patterns []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// FormatFinding returns a human-readable string for a finding.
func FormatFinding(f Finding) string {
	return fmt.Sprintf("%s:%d  [%s]  %s", f.File, f.Line, f.Type, f.Content)
}
