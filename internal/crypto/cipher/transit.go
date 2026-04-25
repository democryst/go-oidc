package cipher

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/openbao/openbao/api/v2"

	"github.com/democryst/go-oidc/pkg/interfaces"
)

type TransitEncryptor struct {
	client  *api.Client
	mount   string
	keyName string
}

func NewTransitEncryptor(client *api.Client, mount, keyName string) interfaces.Encryptor {
	return &TransitEncryptor{client: client, mount: mount, keyName: keyName}
}

func (e *TransitEncryptor) Encrypt(ctx context.Context, data []byte) ([]byte, error) {
	path := fmt.Sprintf("%s/encrypt/%s", e.mount, e.keyName)
	req := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(data),
	}

	secret, err := e.client.Logical().WriteWithContext(ctx, path, req)
	if err != nil {
		return nil, fmt.Errorf("transit encrypt: %w", err)
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("transit encrypt response missing ciphertext")
	}

	return []byte(ciphertext), nil
}

func (e *TransitEncryptor) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	path := fmt.Sprintf("%s/decrypt/%s", e.mount, e.keyName)
	req := map[string]interface{}{
		"ciphertext": string(data),
	}

	secret, err := e.client.Logical().WriteWithContext(ctx, path, req)
	if err != nil {
		return nil, fmt.Errorf("transit decrypt: %w", err)
	}

	plaintextB64, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("transit decrypt response missing plaintext")
	}

	return base64.StdEncoding.DecodeString(plaintextB64)
}
