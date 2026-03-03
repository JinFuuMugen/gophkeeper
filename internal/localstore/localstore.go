package localstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ServerURL     string `json:"server_url"`
	Token         string `json:"token"`
	KDFSaltB64    string `json:"kdf_salt_b64"`
	LastSyncRFC   string `json:"last_sync_rfc3339nano"`
	ConfigVersion int    `json:"config_version"`
}

func EnsureDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}

func configPath(dir string) string { return filepath.Join(dir, "config.json") }
func itemsPath(dir string) string  { return filepath.Join(dir, "items.json") }

func SaveConfig(dir string, cfg Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	final := configPath(dir)
	tmp := final + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	_, err = f.Write(b)
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write tmp: %w", err)
	}

	err = f.Sync()
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("sync tmp: %w", err)
	}

	err = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close tmp: %w", err)
	}

	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename tmp to final: %w", err)
	}
	return nil
}

func LoadConfig(dir string) (Config, error) {
	b, err := os.ReadFile(configPath(dir))
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w (run: gophkeeper login)", err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if cfg.ServerURL == "" || cfg.Token == "" || cfg.KDFSaltB64 == "" {
		return Config{}, fmt.Errorf("invalid config (run: gophkeeper login)")
	}
	return cfg, nil
}
