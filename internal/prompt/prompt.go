package prompt

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// Passphrase prompts for a passphrase without echoing.
func Passphrase(label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", label)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // newline after hidden input
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}
	return string(pass), nil
}

// PassphraseWithConfirm prompts twice and checks they match.
func PassphraseWithConfirm() (string, error) {
	pass1, err := Passphrase("Enter passphrase")
	if err != nil {
		return "", err
	}
	pass2, err := Passphrase("Confirm passphrase")
	if err != nil {
		return "", err
	}
	if pass1 != pass2 {
		return "", fmt.Errorf("passphrases do not match")
	}
	return pass1, nil
}
