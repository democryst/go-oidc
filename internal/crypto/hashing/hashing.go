package hashing

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/sha3"
	
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// SHA3Hasher implements the Hasher interface using SHA3-256.
type SHA3Hasher struct{}

// NewSHA3Hasher creates a new instance of SHA3Hasher.
func NewSHA3Hasher() interfaces.Hasher {
	return &SHA3Hasher{}
}

// sha3Hash calculates the SHA3-256 hash of the input data.
func sha3Hash(data []byte) []byte {
	h := sha3.New256()
	h.Write(data)
	return h.Sum(nil)
}

// base64URLEncode encodes bytes to a Base64URL string without padding.
func base64URLEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// PKCEChallenge calculates the PKCE code challenge from a verifier string.
// It hashes the verifier using SHA3-256 and Base64URL encodes the result.
func (s *SHA3Hasher) PKCEChallenge(verifier string) string {
	hash := sha3Hash([]byte(verifier))
	return base64URLEncode(hash)
}

// VerifyPKCE verifies if the provided challenge matches the expected challenge derived from the verifier.
// It uses constant-time comparison.
func (s *SHA3Hasher) VerifyPKCE(verifier, challenge string) bool {
	// 1. Calculate the expected hash bytes from the verifier.
	expectedHash := sha3Hash([]byte(verifier))

	// 2. Decode the input challenge string (Base64URL) into bytes.
	// We use RawURLEncoding because the input challenge is expected to be unpadded.
	decodedChallenge, err := base64.RawURLEncoding.DecodeString(challenge)
	if err != nil {
		// If decoding fails, the challenge is invalid.
		return false
	}

	// 3. Use constant-time comparison.
	// subtle.ConstantTimeCompare returns 1 if the slices are equal, 0 otherwise.
	if subtle.ConstantTimeCompare(expectedHash, decodedChallenge) == 1 {
		return true
	}
	return false
}

// HashCode calculates the SHA3-256 hash of the raw code and returns the raw bytes.
func (s *SHA3Hasher) HashCode(rawCode string) []byte {
	return sha3Hash([]byte(rawCode))
}

// SecureNonce generates a cryptographically secure random nonce (32 bytes)
// and returns it as a Base64URL encoded string.
func (s *SHA3Hasher) SecureNonce() (string, error) {
	nonce := make([]byte, 32)
	n, err := rand.Read(nonce)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	if n != 32 {
		return "", errors.New("failed to read expected number of random bytes")
	}

	return base64URLEncode(nonce), nil
}
