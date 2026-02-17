package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
)

var errNoJWTSecret = errors.New("no JWT secret provided")
var errNoDatabaseURI = errors.New("no database URI provided")
var errNoMasterKey = errors.New("no master key provided")

type ServerConfig struct {
	Addr         string
	DatabaseURI  string
	JWTSecret    string
	AccessTTL    int
	TLSCertFile  string
	TLSKeyFile   string
	MasterKeyB64 string
}

func LoadServerConfig() (*ServerConfig, error) {
	cfg := ServerConfig{
		Addr:         "localhost:8080",
		DatabaseURI:  "",
		JWTSecret:    "",
		AccessTTL:    15,
		TLSCertFile:  "",
		TLSKeyFile:   "",
		MasterKeyB64: "",
	}

	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database URI")
	flag.StringVar(&cfg.JWTSecret, "jwt", "", "jwt secret")
	flag.IntVar(&cfg.AccessTTL, "attl", 15, "access ttl")
	flag.StringVar(&cfg.TLSCertFile, "tls-cert", "", "path to TLS certificate file ")
	flag.StringVar(&cfg.TLSKeyFile, "tls-key", "", "path to TLS private key file")
	flag.StringVar(&cfg.MasterKeyB64, "master-key", "", "base64 of 32 bytes AES-256 master key")

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

	if v, ok := os.LookupEnv("MASTER_KEY_B64"); ok {
		cfg.MasterKeyB64 = v
	}

	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return nil, fmt.Errorf("cannot load server config: both TLS_CERT_FILE and TLS_KEY_FILE must be set or both empty")
	}

	if cfg.MasterKeyB64 == "" {
		return nil, fmt.Errorf("cannot load server config: %w", errNoMasterKey)
	}

	return &cfg, nil
}
