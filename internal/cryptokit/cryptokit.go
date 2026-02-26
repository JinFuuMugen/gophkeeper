package cryptokit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

var (
	ErrBadKeyLen = errors.New("master key must be 32 bytes")
)

// MasterCrypt encrypts/decrypts using AES-256-GCM
type MasterCrypt struct {
	aead cipher.AEAD
}

func New(key []byte) (*MasterCrypt, error) {
	if len(key) != 32 {
		return nil, ErrBadKeyLen
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return &MasterCrypt{aead: aead}, nil
}

// Encrypt returns nonce||ciphertext
func (m *MasterCrypt) Encrypt(plaintext []byte, aad []byte) ([]byte, error) {
	if m == nil || m.aead == nil {
		return nil, errors.New("cryptokit not initialized")
	}
	nonce := make([]byte, m.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	ciphertext := m.aead.Seal(nil, nonce, plaintext, aad)
	out := make([]byte, 0, len(nonce)+len(ciphertext))
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

func (m *MasterCrypt) Decrypt(in []byte, aad []byte) ([]byte, error) {
	if m == nil || m.aead == nil {
		return nil, errors.New("cryptokit not initialized")
	}
	ns := m.aead.NonceSize()
	if len(in) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce := in[:ns]
	ciphertext := in[ns:]
	plain, err := m.aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plain, nil
}

// KDFParams defines Argon2id settings
type KDFParams struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

func DefaultKDFParams() KDFParams {
	return KDFParams{
		Time:    3,
		Memory:  64 * 1024,
		Threads: 4,
		KeyLen:  32,
	}
}

// DeriveKeyArgon2id derives a 32byte key from master password and salt
func DeriveKeyArgon2id(masterPass []byte, salt []byte, p KDFParams) []byte {
	return argon2.IDKey(masterPass, salt, p.Time, p.Memory, p.Threads, p.KeyLen)
}
