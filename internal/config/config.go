package config

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

type Config struct {
	DiscordToken      string
	DiscordGuildID    string
	RoleVIPID         string
	RoleDefaultID     string
	RoleFreeID        string
	TopAraChannelID   string
	TopArbChannelID   string
	MomentumChannelID string
	LaravelAPIURL     string
	DatabasePath      string
	APIPort           string
}

const (
	DefaultRoleVIPID         = "1546438682276528168"
	DefaultRoleMemberID      = "1546692502785101874"
	DefaultRoleFreeID        = "1546692502785101874"
	DefaultTopAraChannelID   = "1546695838779314226"
	DefaultTopArbChannelID   = "1546695999748050944"
	DefaultMomentumChannelID = "1546695838779314226"
	DefaultLaravelAPIURL     = "http://127.0.0.1:8000/api/v1/agent"
	DefaultDatabasePath      = "nusa.db"
	DefaultAPIPort           = ":8080"
)

var (
	currentConfig *Config
	configLock    sync.RWMutex
)

func init() {
	_ = LoadEnv(".env")
	Load()
}

func Load() *Config {
	configLock.Lock()
	defer configLock.Unlock()

	currentConfig = &Config{
		DiscordToken:      strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		DiscordGuildID:    strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID")),
		RoleVIPID:         firstNonEmpty(os.Getenv("ROLE_VIP_ID"), os.Getenv("DISCORD_ROLE_VIP"), DefaultRoleVIPID),
		RoleDefaultID:     firstNonEmpty(os.Getenv("ROLE_DEFAULT_ID"), os.Getenv("DEFAULT_MEMBER_ROLE_ID"), os.Getenv("DISCORD_ROLE_DEFAULT"), DefaultRoleMemberID),
		RoleFreeID:        firstNonEmpty(os.Getenv("ROLE_FREE_ID"), os.Getenv("FREE_ROLE_ID"), os.Getenv("DISCORD_ROLE_FREE"), DefaultRoleFreeID),
		TopAraChannelID:   firstNonEmpty(os.Getenv("TOP_ARA_CHANNEL_ID"), os.Getenv("DISCORD_CHANNEL_ARA"), DefaultTopAraChannelID),
		TopArbChannelID:   firstNonEmpty(os.Getenv("TOP_ARB_CHANNEL_ID"), os.Getenv("DISCORD_CHANNEL_ARB"), DefaultTopArbChannelID),
		MomentumChannelID: firstNonEmpty(os.Getenv("MOMENTUM_CHANNEL_ID"), os.Getenv("DISCORD_CHANNEL_MOMENTUM"), DefaultMomentumChannelID),
		LaravelAPIURL:     strings.TrimRight(firstNonEmpty(os.Getenv("LARAVEL_API_URL"), DefaultLaravelAPIURL), "/"),
		DatabasePath:      firstNonEmpty(os.Getenv("DATABASE_PATH"), DefaultDatabasePath),
		APIPort:           formatPort(firstNonEmpty(os.Getenv("PORT"), os.Getenv("API_PORT"), DefaultAPIPort)),
	}

	return currentConfig
}

func Get() *Config {
	return Load()
}

func (c *Config) GetRoleForTier(tier string) string {
	switch strings.ToUpper(strings.TrimSpace(tier)) {
	case "VIP":
		return c.RoleVIPID
	default:
		return c.RoleDefaultID
	}
}

func LoadEnv(filenames ...string) error {
	if len(filenames) == 0 {
		filenames = []string{".env"}
	}

	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			line = strings.TrimPrefix(line, "export ")
			idx := strings.Index(line, "=")
			if idx <= 0 {
				continue
			}

			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])

			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}

			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	}

	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formatPort(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ":8080"
	}
	if !strings.HasPrefix(p, ":") {
		return ":" + p
	}
	return p
}
