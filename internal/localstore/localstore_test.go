package localstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JinFuuMugen/GophKeeper/internal/cliapp/clientapi"
)

func TestEnsureDir(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "subdir")
	if err := EnsureDir(nested); err != nil {
		t.Fatalf("EnsureDir returned error: %v", err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("expected directory to exist, stat error: %v", err)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		ServerURL:     "http://example.com",
		Token:         "token123",
		KDFSaltB64:    "c2FsdA==",
		LastSyncRFC:   "2023-01-02T15:04:05Z",
		ConfigVersion: 1,
	}
	if err := SaveConfig(dir, cfg); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}
	loaded, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}
	if loaded != cfg {
		t.Fatalf("expected loaded config to equal saved, got %+v", loaded)
	}
}

func TestLoadConfig_FileNotExist(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadConfig(dir); err == nil {
		t.Fatalf("expected error when loading non-existent config file")
	}
}

func TestLoadConfig_Invalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := configPath(dir)

	if err := os.WriteFile(cfgPath, []byte("{bad json"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := LoadConfig(dir); err == nil {
		t.Fatalf("expected error on invalid JSON")
	}

	badCfg := Config{ServerURL: "", Token: "", KDFSaltB64: ""}
	b, _ := json.Marshal(badCfg)
	if err := os.WriteFile(cfgPath, b, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := LoadConfig(dir); err == nil {
		t.Fatalf("expected error on invalid config content")
	}
}

func newItem(id, typ, meta string) clientapi.Item {
	return clientapi.Item{
		ID:           id,
		Type:         typ,
		EncryptedB64: "ZW5jcnlwdGVk",
		Metadata:     meta,
		Version:      1,
	}
}

func TestReplaceAndLoadItems(t *testing.T) {
	dir := t.TempDir()
	items := []clientapi.Item{
		newItem("id1", "text", "meta1"),
		newItem("id2", "file", "meta2"),
	}
	if err := ReplaceLocalItems(dir, items); err != nil {
		t.Fatalf("ReplaceLocalItems returned error: %v", err)
	}
	loaded, err := LoadLocalItems(dir)
	if err != nil {
		t.Fatalf("LoadLocalItems returned error: %v", err)
	}
	if len(loaded) != len(items) {
		t.Fatalf("expected %d items, got %d", len(items), len(loaded))
	}
}

func TestLoadLocalItems_NoFile(t *testing.T) {
	dir := t.TempDir()
	items, err := LoadLocalItems(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected nil or empty slice on missing items file")
	}
}

func TestUpsertLocalItem(t *testing.T) {
	dir := t.TempDir()

	it := newItem("id1", "text", "meta1")
	if err := UpsertLocalItem(dir, it); err != nil {
		t.Fatalf("UpsertLocalItem returned error: %v", err)
	}
	items, err := LoadLocalItems(dir)
	if err != nil {
		t.Fatalf("LoadLocalItems returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "id1" {
		t.Fatalf("expected one item with ID id1, got %+v", items)
	}

	it2 := newItem("id1", "text", "meta2")
	if err := UpsertLocalItem(dir, it2); err != nil {
		t.Fatalf("UpsertLocalItem returned error: %v", err)
	}
	items, err = LoadLocalItems(dir)
	if err != nil {
		t.Fatalf("LoadLocalItems returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one item after update, got %d", len(items))
	}
	if strings.TrimSpace(items[0].Metadata) != "meta2" {
		t.Fatalf("expected metadata to be updated to meta2, got %s", items[0].Metadata)
	}
}

func TestUpsertLocalItem_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(itemsPath(dir), []byte("{bad json"), 0o600); err != nil {
		t.Fatalf("write items: %v", err)
	}
	if err := UpsertLocalItem(dir, newItem("id1", "text", "meta")); err == nil {
		t.Fatalf("expected error when upserting with invalid items file")
	}
}
