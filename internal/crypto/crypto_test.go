package crypto

import (
	"testing"
)

func TestDeriveKeyDeterminism(t *testing.T) {
	salt := []byte("testsalt12345678")
	key1 := DeriveKey("mypassphrase", salt)
	key2 := DeriveKey("mypassphrase", salt)

	if len(key1) != KeyLen {
		t.Fatalf("expected key length %d, got %d", KeyLen, len(key1))
	}
	for i := range key1 {
		if key1[i] != key2[i] {
			t.Fatal("same passphrase and salt should produce identical keys")
		}
	}
}

func TestDeriveKeyDifferentPassphrase(t *testing.T) {
	salt := []byte("testsalt12345678")
	key1 := DeriveKey("passphrase1", salt)
	key2 := DeriveKey("passphrase2", salt)

	same := true
	for i := range key1 {
		if key1[i] != key2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different passphrases should produce different keys")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := DeriveKey("testpass", []byte("testsalt12345678"))
	plaintext := []byte("DB_HOST=localhost\nDB_PASS=secret123")

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestTamperedCiphertextDetection(t *testing.T) {
	key := DeriveKey("testpass", []byte("testsalt12345678"))
	encrypted, err := Encrypt([]byte("secret"), key)
	if err != nil {
		t.Fatal(err)
	}

	// Tamper with the ciphertext
	tampered := []byte(encrypted)
	tampered[len(tampered)-2] ^= 0xFF
	_, err = Decrypt(string(tampered), key)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestWrongKeyError(t *testing.T) {
	key1 := DeriveKey("correctpass", []byte("testsalt12345678"))
	key2 := DeriveKey("wrongpass", []byte("testsalt12345678"))

	encrypted, err := Encrypt([]byte("secret"), key1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(encrypted, key2)
	if err != ErrWrongPassphrase {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) != SaltLen {
		t.Fatalf("expected salt length %d, got %d", SaltLen, len(salt))
	}
}
