package config

import (
	"log"
	"os"
	"strings"
)

func MustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("missing required env: %s", key)
	}
	return v
}

func GetEnvDefault(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}
