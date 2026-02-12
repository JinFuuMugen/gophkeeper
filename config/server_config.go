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

type ServerConfig struct {
	Addr        string
	DatabaseURI string
	JWTSecret   string
	AccessTTL   int
	// ShutdownWait time.Duration
}

func LoadServerConfig() (*ServerConfig, error) {
	cfg := ServerConfig{
		Addr:        "localhost:8080",
		DatabaseURI: "",
		JWTSecret:   "",
		AccessTTL:   15,
		// ShutdownWait: 10 * time.Second,
	}

	flag.StringVar(&cfg.Addr, "a", "localhost:8080", "server address")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "database URI")
	flag.StringVar(&cfg.JWTSecret, "jwt", "", "jwt secret")
	flag.IntVar(&cfg.AccessTTL, "attl", 15, "access ttl")
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

	return &cfg, nil
}
