package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

/**
 * Symmetric:
 * 1. same key for encrypt and decrypt.
 * 2. Fast.
 * 3. AES (Advanced Encryption Standard) is here.
 *
 * Asymmetric:
 * 1. public key encrypts,
 * 2. private key decrypts.
 * 3. Slower.
 * 4. RSA, elliptic curves are here.
 *
 * 16 bytes -> AES-128
 * 24 bytes -> AES-192
 * 32 bytes -> AES-256
 */
func Encrypt(key, payload []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	/* gcm.Seal appends cipher text + tag to nonce */
	sealed := gcm.Seal(nonce, nonce, payload, nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func Decrypt(key []byte, encoded string) ([]byte, error) {
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	ns := gcm.NonceSize()
	if len(data) < ns {
		return nil, fmt.Errorf("cipher text too short")
	}

	plaintext, err := gcm.Open(nil, data[:ns], data[ns:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	return plaintext, nil
}
