package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_Defaults(t *testing.T) {
	for _, k := range []string{"APP_ENV", "PORT", "DATABASE_URL", "JWT_SECRET", "JWT_EXPIRY_HOURS", "ALLOWED_ORIGINS"} {
		t.Setenv(k, "")
	}
	c := Load()
	assert.Equal(t, EnvLocal, c.AppEnv)
	assert.Equal(t, "8080", c.Port)
	assert.Contains(t, c.DatabaseURL, "localhost:5432/neonflow")
	assert.Equal(t, 12, c.JWTExpiryHours)
	assert.Equal(t, []string{"http://localhost:5173"}, c.AllowedOrigins)
	assert.NoError(t, c.Validate())
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PORT", "9000")
	t.Setenv("JWT_EXPIRY_HOURS", "abc")
	t.Setenv("ALLOWED_ORIGINS", " https://a.app , ,https://b.app ")
	c := Load()
	assert.True(t, c.IsDeployed())
	assert.Equal(t, "9000", c.Port)
	assert.Equal(t, 12, c.JWTExpiryHours, "valor inválido cai no default")
	assert.Equal(t, []string{"https://a.app", "https://b.app"}, c.AllowedOrigins)
}

func TestIsDeployed(t *testing.T) {
	assert.False(t, Configs{AppEnv: EnvLocal}.IsDeployed())
	for _, env := range []string{EnvHml, EnvProduction, "staging"} {
		assert.True(t, Configs{AppEnv: env}.IsDeployed(), env)
	}
}

func TestValidate_Deployed(t *testing.T) {
	for _, env := range []string{EnvHml, EnvProduction, "staging"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)

			t.Setenv("JWT_SECRET", "")
			t.Setenv("DATABASE_URL", "postgres://x")
			assert.ErrorIs(t, Load().Validate(), ErrMissingJWTSecret)

			t.Setenv("JWT_SECRET", devJWTSecret)
			assert.ErrorIs(t, Load().Validate(), ErrMissingJWTSecret, "segredo de desenvolvimento é recusado")

			t.Setenv("JWT_SECRET", "um-segredo-de-verdade")
			t.Setenv("DATABASE_URL", "")
			assert.ErrorIs(t, Load().Validate(), ErrMissingDatabaseURL)

			t.Setenv("DATABASE_URL", "postgres://x")
			assert.NoError(t, Load().Validate())
		})
	}
}
