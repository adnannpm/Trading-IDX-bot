package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	// Clear relevant env vars
	envVars := []string{
		"DISCORD_TOKEN", "DISCORD_GUILD_ID", "ROLE_VIP_ID", "DISCORD_ROLE_VIP",
		"ROLE_DEFAULT_ID", "DEFAULT_MEMBER_ROLE_ID", "DISCORD_ROLE_DEFAULT",
		"ROLE_FREE_ID", "FREE_ROLE_ID", "DISCORD_ROLE_FREE",
		"TOP_ARA_CHANNEL_ID", "DISCORD_CHANNEL_ARA",
		"TOP_ARB_CHANNEL_ID", "DISCORD_CHANNEL_ARB",
		"LARAVEL_API_URL", "DATABASE_PATH", "PORT", "API_PORT",
	}
	for _, k := range envVars {
		os.Unsetenv(k)
	}

	cfg := Load()
	if cfg.RoleVIPID != DefaultRoleVIPID {
		t.Errorf("expected RoleVIPID %s, got %s", DefaultRoleVIPID, cfg.RoleVIPID)
	}
	if cfg.RoleDefaultID != DefaultRoleMemberID {
		t.Errorf("expected RoleDefaultID %s, got %s", DefaultRoleMemberID, cfg.RoleDefaultID)
	}
	if cfg.RoleFreeID != DefaultRoleFreeID {
		t.Errorf("expected RoleFreeID %s, got %s", DefaultRoleFreeID, cfg.RoleFreeID)
	}
	if cfg.TopAraChannelID != DefaultTopAraChannelID {
		t.Errorf("expected TopAraChannelID %s, got %s", DefaultTopAraChannelID, cfg.TopAraChannelID)
	}
	if cfg.TopArbChannelID != DefaultTopArbChannelID {
		t.Errorf("expected TopArbChannelID %s, got %s", DefaultTopArbChannelID, cfg.TopArbChannelID)
	}
	if cfg.LaravelAPIURL != DefaultLaravelAPIURL {
		t.Errorf("expected LaravelAPIURL %s, got %s", DefaultLaravelAPIURL, cfg.LaravelAPIURL)
	}
	if cfg.DatabasePath != DefaultDatabasePath {
		t.Errorf("expected DatabasePath %s, got %s", DefaultDatabasePath, cfg.DatabasePath)
	}
	if cfg.APIPort != DefaultAPIPort {
		t.Errorf("expected APIPort %s, got %s", DefaultAPIPort, cfg.APIPort)
	}
}

func TestConfigOverrides(t *testing.T) {
	os.Setenv("DISCORD_TOKEN", "test-token-123")
	os.Setenv("ROLE_VIP_ID", "vip-999")
	os.Setenv("TOP_ARA_CHANNEL_ID", "ara-888")
	os.Setenv("API_PORT", "9090")
	defer func() {
		os.Unsetenv("DISCORD_TOKEN")
		os.Unsetenv("ROLE_VIP_ID")
		os.Unsetenv("TOP_ARA_CHANNEL_ID")
		os.Unsetenv("API_PORT")
	}()

	cfg := Load()
	if cfg.DiscordToken != "test-token-123" {
		t.Errorf("expected test-token-123, got %s", cfg.DiscordToken)
	}
	if cfg.RoleVIPID != "vip-999" {
		t.Errorf("expected vip-999, got %s", cfg.RoleVIPID)
	}
	if cfg.TopAraChannelID != "ara-888" {
		t.Errorf("expected ara-888, got %s", cfg.TopAraChannelID)
	}
	if cfg.APIPort != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.APIPort)
	}

	if cfg.GetRoleForTier("VIP") != "vip-999" {
		t.Errorf("expected VIP role vip-999, got %s", cfg.GetRoleForTier("VIP"))
	}
	if cfg.GetRoleForTier("REGULAR") != cfg.RoleDefaultID {
		t.Errorf("expected default role for REGULAR, got %s", cfg.GetRoleForTier("REGULAR"))
	}
}

func TestLoadEnv(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env.test")

	content := `
# Comment line
export DISCORD_TOKEN="test-env-token"
ROLE_VIP_ID='role-from-env'
TOP_ARA_CHANNEL_ID=11223344
API_PORT=3000
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test env: %v", err)
	}

	os.Unsetenv("DISCORD_TOKEN")
	os.Unsetenv("ROLE_VIP_ID")
	os.Unsetenv("TOP_ARA_CHANNEL_ID")
	os.Unsetenv("API_PORT")

	if err := LoadEnv(envPath); err != nil {
		t.Fatalf("LoadEnv failed: %v", err)
	}

	if os.Getenv("DISCORD_TOKEN") != "test-env-token" {
		t.Errorf("expected test-env-token, got %s", os.Getenv("DISCORD_TOKEN"))
	}
	if os.Getenv("ROLE_VIP_ID") != "role-from-env" {
		t.Errorf("expected role-from-env, got %s", os.Getenv("ROLE_VIP_ID"))
	}
	if os.Getenv("TOP_ARA_CHANNEL_ID") != "11223344" {
		t.Errorf("expected 11223344, got %s", os.Getenv("TOP_ARA_CHANNEL_ID"))
	}
	if os.Getenv("API_PORT") != "3000" {
		t.Errorf("expected 3000, got %s", os.Getenv("API_PORT"))
	}

	// Clean up
	os.Unsetenv("DISCORD_TOKEN")
	os.Unsetenv("ROLE_VIP_ID")
	os.Unsetenv("TOP_ARA_CHANNEL_ID")
	os.Unsetenv("API_PORT")
}
