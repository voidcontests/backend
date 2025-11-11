package crypto

import (
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	encryptor := NewCipher("test-encryption-key-12345")

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "simple string",
			plaintext: "hello world",
		},
		{
			name:      "mnemonic phrase",
			plaintext: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
		},
		{
			name:      "long string",
			plaintext: strings.Repeat("a", 1000),
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-={}[]|\\:\";<>?,./",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := encryptor.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if ciphertext == "" {
				t.Fatal("Encrypt() returned empty ciphertext")
			}

			if ciphertext == tt.plaintext {
				t.Fatal("Encrypt() returned plaintext unchanged")
			}

			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %v, want %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptEmptyString(t *testing.T) {
	encryptor := NewCipher("test-key")

	_, err := encryptor.Encrypt("")
	if err == nil {
		t.Error("Encrypt() should return error for empty string")
	}
}

func TestDecryptEmptyString(t *testing.T) {
	encryptor := NewCipher("test-key")

	_, err := encryptor.Decrypt("")
	if err == nil {
		t.Error("Decrypt() should return error for empty string")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	encryptor := NewCipher("test-key")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{
			name:       "invalid base64",
			ciphertext: "not-valid-base64!@#",
		},
		{
			name:       "too short",
			ciphertext: "YWJj",
		},
		{
			name:       "random data",
			ciphertext: "SGVsbG8gV29ybGQ=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encryptor.Decrypt(tt.ciphertext)
			if err == nil {
				t.Error("Decrypt() should return error for invalid ciphertext")
			}
		})
	}
}

func TestDifferentKeys(t *testing.T) {
	encryptor1 := NewCipher("key1")
	encryptor2 := NewCipher("key2")

	plaintext := "secret message"

	ciphertext, err := encryptor1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = encryptor2.Decrypt(ciphertext)
	if err == nil {
		t.Error("Decrypt() should fail when using different key")
	}
}

func TestEncryptionUniqueness(t *testing.T) {
	encryptor := NewCipher("test-key")
	plaintext := "same message"

	ciphertext1, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	ciphertext2, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if ciphertext1 == ciphertext2 {
		t.Error("Encrypt() should produce different ciphertexts for same plaintext (different nonces)")
	}

	decrypted1, _ := encryptor.Decrypt(ciphertext1)
	decrypted2, _ := encryptor.Decrypt(ciphertext2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}
