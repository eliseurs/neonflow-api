package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const (
	EnvLocal      = "local"
	EnvHml        = "hml"
	EnvProduction = "production"

	// devJWTSecret só existe para desenvolvimento local. Fora dele a API não sobe com ele.
	devJWTSecret = "neonflow-dev-secret-nao-usar-em-producao"
)

var (
	ErrMissingJWTSecret   = errors.New("JWT_SECRET é obrigatório fora do ambiente local")
	ErrMissingDatabaseURL = errors.New("DATABASE_URL é obrigatório fora do ambiente local")
)

type Configs struct {
	AppEnv         string
	Port           string
	DatabaseURL    string
	JWTSecret      string
	JWTExpiryHours int
	AllowedOrigins []string
	Storage        Storage
}

// Storage é o armazenamento de arquivos compatível com S3 (bucket do Railway hoje, Cloudflare R2 depois;
// ADR-0014). Sem as cinco variáveis, o envio de fotos responde que o armazenamento não está configurado.
type Storage struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

// Enabled diz se todas as variáveis do armazenamento estão definidas.
func (s Storage) Enabled() bool {
	return s.Endpoint != "" && s.Bucket != "" && s.AccessKey != "" && s.SecretKey != ""
}

// Load lê as variáveis de ambiente. Os defaults servem ao ambiente local
// (docker compose); não crie .env.local.
func Load() Configs {
	return Configs{
		AppEnv:         getEnv("APP_ENV", EnvLocal),
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://neonflow:neonflow@localhost:5432/neonflow?sslmode=disable"),
		JWTSecret:      getEnv("JWT_SECRET", devJWTSecret),
		JWTExpiryHours: getEnvInt("JWT_EXPIRY_HOURS", 12),
		AllowedOrigins: splitList(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		Storage: Storage{
			Endpoint:  getEnv("STORAGE_ENDPOINT", ""),
			Region:    getEnv("STORAGE_REGION", "auto"),
			Bucket:    getEnv("STORAGE_BUCKET", ""),
			AccessKey: getEnv("STORAGE_ACCESS_KEY_ID", ""),
			SecretKey: getEnv("STORAGE_SECRET_ACCESS_KEY", ""),
		},
	}
}

// IsDeployed vale para todo ambiente publicado (hml, production ou qualquer
// outro nome): só "local" usa os defaults de desenvolvimento.
func (c Configs) IsDeployed() bool {
	return c.AppEnv != EnvLocal
}

// Validate impede que a API suba fora do ambiente local com defaults de desenvolvimento.
func (c Configs) Validate() error {
	if !c.IsDeployed() {
		return nil
	}
	if c.JWTSecret == "" || c.JWTSecret == devJWTSecret {
		return ErrMissingJWTSecret
	}
	if os.Getenv("DATABASE_URL") == "" {
		return ErrMissingDatabaseURL
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
