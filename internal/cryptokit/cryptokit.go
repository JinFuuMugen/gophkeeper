package cryptokit

// import (
// 	"crypto/aes"
// 	"crypto/cipher"
// 	"crypto/rand"
// 	"encoding/base64"
// 	"errors"
// 	"fmt"
// 	"io"
// )

// var (
// 	ErrBadKeyLen = errors.New("master key must be 32 bytes")
// )

// type MasterCrypt struct {
// 	aead cipher.AEAD
// }

// func NewFromBase64(b64 string) (*MasterCrypt, error) {
// 	raw, err := base64.StdEncoding.DecodeString(b64)
// 	if err != nil {
// 		return nil, fmt.Errorf("decode master key base64: %w", err)
// 	}
// 	return New(raw)
// }

// func New(key []byte) (*MasterCrypt, error) {
// 	if len(key) != 32 {
// 		return nil, ErrBadKeyLen
// 	}

// 	block, err := aes.NewCipher(key)
// 	if err != nil {
// 		return nil, fmt.Errorf("aes cipher: %w", err)
// 	}

// 	aead, err := cipher.NewGCM(block)
// 	if err != nil {
// 		return nil, fmt.Errorf("gcm: %w", err)
// 	}
// 	return &MasterCrypt{aead: aead}, nil
// }

// func (m *MasterCrypt) Encrypt(plaintext []byte, aad []byte) ([]byte, error) {
// 	if m == nil || m.aead == nil {
// 		return nil, errors.New("cryptokit not initialized")
// 	}

// 	nonce := make([]byte, m.aead.NonceSize())
// 	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
// 		return nil, fmt.Errorf("nonce: %w", err)
// 	}

// 	ciphertext := m.aead.Seal(nil, nonce, plaintext, aad)

// 	out := make([]byte, 0, len(nonce)+len(ciphertext))
// 	out = append(out, nonce...)
// 	out = append(out, ciphertext...)
// 	return out, nil
// }

// func (m *MasterCrypt) Decrypt(in []byte, aad []byte) ([]byte, error) {
// 	if m == nil || m.aead == nil {
// 		return nil, errors.New("cryptokit not initialized")
// 	}

// 	ns := m.aead.NonceSize()
// 	if len(in) < ns {
// 		return nil, errors.New("ciphertext too short")
// 	}
// 	nonce := in[:ns]
// 	ciphertext := in[ns:]

// 	plain, err := m.aead.Open(nil, nonce, ciphertext, aad)
// 	if err != nil {
// 		return nil, fmt.Errorf("decrypt: %w", err)
// 	}
// 	return plain, nil
// }
