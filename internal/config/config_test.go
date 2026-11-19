package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Defaults(t *testing.T) {
	keys := []string{
		"APP_ENV", "APP_PORT", "APP_KEY", "ADMIN_USERNAME", "ADMIN_PASSWORD",
		"DB_CONNECTION", "DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME",
		"DB_PASSWORD", "DB_SSLMODE", "SEARCH_ENGINE_ENDPOINT", "OPENAI_ENDPOINT",
		"OPENAI_KEY", "OPENAI_MODEL",
	}
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}

	cfg := LoadConfig()
	assert.Equal(t, "development", cfg.AppEnv)
	assert.Equal(t, "8081", cfg.AppPort)
	assert.Equal(t, "delcom_blog_secret_key_super_secure_2026_random_string_32chars", cfg.AppKey)
	assert.Equal(t, "admin", cfg.AdminUsername)
	assert.Equal(t, "admin", cfg.AdminPassword)
	assert.Equal(t, "pgsql", cfg.DBConnection)
	assert.Equal(t, "127.0.0.1", cfg.DBHost)
	assert.Equal(t, "5432", cfg.DBPort)
	assert.Equal(t, "pabwe_p9_blog", cfg.DBDatabase)
	assert.Equal(t, "postgres", cfg.DBUsername)
	assert.Equal(t, "postgres", cfg.DBPassword)
	assert.Equal(t, "disable", cfg.DBSSLMode)
	assert.Equal(t, "https://delcom.org/api/v1/search", cfg.SearchEngineEndpoint)
	assert.Equal(t, "https://delcom.org/api/v1", cfg.OpenAIEndpoint)
	assert.Equal(t, "", cfg.OpenAIKey)
	assert.Equal(t, "qwen3.8-flash", cfg.OpenAIModel)
}

func TestLoadConfig_FromEnvFileAndEnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envFile, []byte("APP_PORT=9090\nOPENAI_KEY=test-key-123\n"), 0644)
	assert.NoError(t, err)

	_ = os.Unsetenv("APP_PORT")
	_ = os.Unsetenv("OPENAI_KEY")

	cfg := LoadConfig(envFile)
	assert.Equal(t, "9090", cfg.AppPort)
	assert.Equal(t, "test-key-123", cfg.OpenAIKey)

	_ = os.Unsetenv("APP_PORT")
	_ = os.Unsetenv("OPENAI_KEY")
}
