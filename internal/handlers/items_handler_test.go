package handlers_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/JinFuuMugen/GophKeeper/internal/handlers"
	"github.com/JinFuuMugen/GophKeeper/internal/models"
	"github.com/google/uuid"
)

func TestToItemResp_NotDeleted_WithEncrypted(t *testing.T) {
	id := uuid.New()
	now := time.Unix(0, 0)
	enc := []byte{1, 2, 3}
	wantB64 := base64.StdEncoding.EncodeToString(enc)

	resp := handlers.ToItemResp(models.Item{
		ID:        id,
		Type:      "text",
		Encrypted: enc,
		Metadata:  "m",
		Version:   2,
		Deleted:   false,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if resp.ID != id.String() {
		t.Fatalf("expected id %s, got %s", id.String(), resp.ID)
	}
	if resp.EncryptedB64 != wantB64 {
		t.Fatalf("expected encrypted_b64 %s, got %s", wantB64, resp.EncryptedB64)
	}
}

func TestToItemResp_Deleted_NoEncrypted(t *testing.T) {
	id := uuid.New()
	now := time.Unix(0, 0)

	resp := handlers.ToItemResp(models.Item{
		ID:        id,
		Type:      "text",
		Encrypted: []byte{9, 9, 9},
		Metadata:  "m",
		Version:   1,
		Deleted:   true,
		CreatedAt: now,
		UpdatedAt: now,
	})

	if resp.ID != id.String() {
		t.Fatalf("expected id %s, got %s", id.String(), resp.ID)
	}
	if resp.EncryptedB64 != "" {
		t.Fatalf("expected empty encrypted_b64 when deleted, got %s", resp.EncryptedB64)
	}
}
