package config_test

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/JinFuuMugen/GophKeeper/config"
	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
)

func TestValidateConfig(t *testing.T) {
	if err := config.ValidateConfig(&config.ServerConfig{}); err != errdefs.ErrBadConfigValue {
		t.Fatalf("expected ErrBadConfigValue, got %v", err)
	}
	if err := config.ValidateConfig(&config.ServerConfig{Addr: "localhost:8080"}); err != errdefs.ErrBadConfigValue {
		t.Fatalf("expected ErrBadConfigValue, got %v", err)
	}
	valid := &config.ServerConfig{
		Addr:        "localhost:8080",
		DatabaseURI: "postgres://user:pass@localhost/db",
		JWTSecret:   "secret",
		AccessTTL:   15,
	}
	if err := config.ValidateConfig(valid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadServerConfig(t *testing.T) {
	origEnv := os.Environ()
	origArgs := os.Args

	restore := func() {
		os.Clearenv()
		for _, kv := range origEnv {
			parts := strings.SplitN(kv, "=", 2)
			_ = os.Setenv(parts[0], parts[1])
		}
		os.Args = origArgs
	}
	defer restore()

	os.Args = []string{origArgs[0]}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Clearenv()

	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatalf("expected error due to missing JWT_SECRET")
	}

	os.Setenv("JWT_SECRET", "secret")
	os.Args = []string{origArgs[0]}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatalf("expected error due to missing DATABASE_URI")
	}

	os.Setenv("DATABASE_URI", "postgres://user:pass@localhost/db")
	os.Args = []string{origArgs[0]}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cfg, err := config.LoadServerConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.JWTSecret != "secret" || cfg.DatabaseURI != "postgres://user:pass@localhost/db" {
		t.Fatalf("env variables not loaded")
	}

	os.Setenv("TLS_CERT_FILE", "/path/to/cert")
	os.Unsetenv("TLS_KEY_FILE")
	os.Args = []string{origArgs[0]}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	if _, err := config.LoadServerConfig(); err == nil {
		t.Fatalf("expected error when only one TLS env var set")
	}
}
