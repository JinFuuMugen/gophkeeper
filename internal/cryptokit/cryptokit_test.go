package cryptokit

import (
	"bytes"
	"testing"
)

func TestNew_BadKeyLen(t *testing.T) {
	if _, err := New([]byte("short")); err == nil {
		t.Fatalf("expected error for short key")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	kit, err := New(key)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	plain := []byte("hello world")
	aad := []byte("aad")

	ct, err := kit.Encrypt(plain, aad)
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}
	out, err := kit.Decrypt(ct, aad)
	if err != nil {
		t.Fatalf("Decrypt returned error: %v", err)
	}
	if !bytes.Equal(out, plain) {
		t.Fatalf("expected decrypted plaintext %q, got %q", plain, out)
	}
}

func TestDecrypt_WrongAAD(t *testing.T) {
	key := bytes.Repeat([]byte{2}, 32)
	kit, err := New(key)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	plain := []byte("data")
	ct, err := kit.Encrypt(plain, []byte("aad1"))
	if err != nil {
		t.Fatalf("Encrypt returned error: %v", err)
	}

	if _, err := kit.Decrypt(ct, []byte("aad2")); err == nil {
		t.Fatalf("expected error when decrypting with wrong AAD")
	}
}

func TestDecrypt_ShortCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{3}, 32)
	kit, err := New(key)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if _, err := kit.Decrypt([]byte{1, 2, 3}, []byte{}); err == nil {
		t.Fatalf("expected error on too short ciphertext")
	}
}
