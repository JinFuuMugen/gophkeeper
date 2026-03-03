package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/JinFuuMugen/GophKeeper/internal/errdefs"
)

var errNoJWTSecret = errors.New("no JWT secret provided")
var errNoDatabaseURI = errors.New("no database URI provided")

type ServerConfig struct {
	Addr        string
	DatabaseURI string
	JWTSecret   string
	AccessTTL   int
	TLSCertFile string
	TLSKeyFile  string
}

func ValidateConfig(cfg *ServerConfig) error {
	if cfg.Addr == "" || cfg.AccessTTL <= 0 || cfg.JWTSecret == "" || cfg.DatabaseURI == "" {
		return errdefs.ErrBadConfigValue
	}

	return nil
}

func LoadServerConfig() (*ServerConfig, error) {
	cfg := ServerConfig{
		Addr:        "localhost:8080",
		DatabaseURI: "",
		JWTSecret:   "",
		AccessTTL:   15,
		TLSCertFile: "",
		TLSKeyFile:  "",
	}

	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database URI")
	flag.StringVar(&cfg.JWTSecret, "jwt", "", "jwt secret")
	flag.IntVar(&cfg.AccessTTL, "attl", 15, "access ttl")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", "", "path to TLS certificate file ")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", "", "path to TLS private key file")

	flag.Parse()

	if v, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = v
	}

	if v, ok := os.LookupEnv("DATABASE_URI"); ok {
		cfg.DatabaseURI = v
	}

	if v, ok := os.LookupEnv("JWT_SECRET"); ok {
		cfg.JWTSecret = v
	}

	if v, ok := os.LookupEnv("ACCESS_TTL"); ok {
		vInt, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("bad ACCESS_TTL value: %w", err)
		}

		cfg.AccessTTL = vInt
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("cannot load server config: %w", errNoJWTSecret)
	}

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("cannot load server config: %w", errNoDatabaseURI)
	}

	if v, ok := os.LookupEnv("TLS_CERT_FILE"); ok {
		cfg.TLSCertFile = v
	}

	if v, ok := os.LookupEnv("TLS_KEY_FILE"); ok {
		cfg.TLSKeyFile = v
	}

	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return nil, fmt.Errorf("cannot load server config: both TLS_CERT_FILE and TLS_KEY_FILE must be set or both empty")
	}

	return &cfg, nil
}
