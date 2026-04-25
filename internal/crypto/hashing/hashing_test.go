package hashing

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestPKCEChallenge(t *testing.T) {
	hasher := NewSHA3Hasher()

	// Given a known test verifier
	verifier := "my-secret-pkce-verifier-1234567890" // 36 bytes

	// Then the challenge should be mathematically derived
	expectedChallenge := base64URLEncode(sha3Hash([]byte(verifier)))
	actualChallenge := hasher.PKCEChallenge(verifier)

	if actualChallenge != expectedChallenge {
		t.Errorf("Expected challenge %s, got %s", expectedChallenge, actualChallenge)
	}

	// It must be unpadded Base64URL
	for _, char := range actualChallenge {
		if char == '+' || char == '/' || char == '=' {
			t.Errorf("Challenge should be Base64-URL unpadded, but found character: %c", char)
		}
	}
}

func TestVerifyPKCE(t *testing.T) {
	hasher := NewSHA3Hasher()

	validVerifier := "valid_code_123"
	validChallenge := hasher.PKCEChallenge(validVerifier)

	tests := []struct {
		name      string
		verifier  string
		challenge string
		expected  bool
	}{
		{"Valid Pair", validVerifier, validChallenge, true},
		{"Invalid Verifier", "wrong_code", validChallenge, false},
		{"Invalid Challenge", validVerifier, "definitely_not_the_challenge", false},
		{"Empty Verifier", "", validChallenge, false},
		{"Empty Challenge", validVerifier, "", false},
		{"Invalid Base64", validVerifier, "not_valid_b64^^%%!!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasher.VerifyPKCE(tt.verifier, tt.challenge)
			if result != tt.expected {
				t.Errorf("VerifyPKCE(%q, %q) = %t; want %t", tt.verifier, tt.challenge, result, tt.expected)
			}
		})
	}
}

func TestHashCode(t *testing.T) {
	hasher := NewSHA3Hasher()

	const input = "some_auth_code_12345"
	expected := sha3Hash([]byte(input))

	actual := hasher.HashCode(input)

	if !bytes.Equal(actual, expected) {
		t.Errorf("HashCode failed. Expected %x, got %x", expected, actual)
	}
}

func TestSecureNonce(t *testing.T) {
	hasher := NewSHA3Hasher()

	nonce1, err := hasher.SecureNonce()
	if err != nil {
		t.Fatalf("SecureNonce failed: %v", err)
	}
	if len(nonce1) == 0 {
		t.Fatal("SecureNonce returned empty string")
	}

	nonce2, err := hasher.SecureNonce()
	if err != nil {
		t.Fatalf("SecureNonce failed: %v", err)
	}

	if nonce1 == nonce2 {
		t.Fatal("SecureNonce generated identical nonces; entropy source failed")
	}

	// Verify it decodes to 32 bytes
	decoded, err := base64.RawURLEncoding.DecodeString(nonce1)
	if err != nil {
		t.Fatalf("Failed to decode nonce: %v", err)
	}
	if len(decoded) != 32 {
		t.Errorf("Expected 32 bytes of entropy, got %d", len(decoded))
	}
}
