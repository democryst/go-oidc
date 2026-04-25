package cipher

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/openbao/openbao/api/v2"
	"github.com/democryst/go-oidc/pkg/interfaces"
)

// OpenBaoEncryptor implements the Encryptor interface using the OpenBao Transit secrets engine.
type OpenBaoEncryptor struct {
	client    *api.Client
	mountPath string
	keyName   string
}

// NewOpenBaoEncryptor creates a new OpenBaoEncryptor instance.
func NewOpenBaoEncryptor(client *api.Client, mountPath, keyName string) interfaces.Encryptor {
	if client == nil {
		return nil
	}
	if mountPath == "" {
		mountPath = "transit"
	}

	return &OpenBaoEncryptor{
		client:    client,
		mountPath: mountPath,
		keyName:   keyName,
	}
}

// Encrypt encrypts the given plaintext using the configured OpenBao key.
func (o *OpenBaoEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	if o.client == nil {
		return nil, fmt.Errorf("openbao client is not initialized")
	}
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}

	path := fmt.Sprintf("%s/encrypt/%s", o.mountPath, o.keyName)
	data := map[string]interface{}{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}

	secret, err := o.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return nil, fmt.Errorf("openbao transit encrypt failed: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("openbao returned empty secret for encryption")
	}

	ciphertextStr, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("openbao response missing ciphertext")
	}

	return []byte(ciphertextStr), nil
}

// Decrypt decrypts the given ciphertext using the configured OpenBao key.
func (o *OpenBaoEncryptor) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if o.client == nil {
		return nil, fmt.Errorf("openbao client is not initialized")
	}
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("ciphertext cannot be empty")
	}

	path := fmt.Sprintf("%s/decrypt/%s", o.mountPath, o.keyName)
	data := map[string]interface{}{
		"ciphertext": string(ciphertext),
	}

	secret, err := o.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return nil, fmt.Errorf("openbao transit decrypt failed: %w", err)
	}
	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("openbao returned empty secret for decryption")
	}

	plaintextB64, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("openbao response missing plaintext")
	}

	plaintext, err := base64.StdEncoding.DecodeString(plaintextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 plaintext: %w", err)
	}

	return plaintext, nil
}
