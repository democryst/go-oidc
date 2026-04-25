package signer

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/cloudflare/circl/sign/dilithium/mode3"
	"github.com/google/uuid"

	"github.com/democryst/go-oidc/pkg/interfaces"
)

// PQCConfig holds IDs for the fallback keys.
type PQCConfig struct {
	DilithiumKeyID uuid.UUID
}

// CreatePQCFetcher returns a function that retrieves and decrypts a Dilithium3 key.
func CreatePQCFetcher(repo interfaces.Repository, decryptor interfaces.Encryptor, keyID uuid.UUID) func(context.Context) (*mode3.PrivateKey, error) {
	return func(ctx context.Context) (*mode3.PrivateKey, error) {
		algo, encHex, err := repo.GetPQCKey(ctx, keyID)
		if err != nil {
			return nil, err
		}
		if algo != "Dilithium3" {
			return nil, fmt.Errorf("unsupported PQC algorithm: %s", algo)
		}

		encHexBytes, err := hex.DecodeString(encHex)
		if err != nil {
			return nil, fmt.Errorf("invalid key hex: %w", err)
		}

		decBytes, err := decryptor.Decrypt(ctx, encHexBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt pqc key: %w", err)
		}

		priv := new(mode3.PrivateKey)
		err = priv.UnmarshalBinary(decBytes)
		return priv, err
	}
}
