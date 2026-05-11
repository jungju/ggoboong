package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

const (
	DefaultAppID      int64 = 3675420
	DefaultConfigName       = "ggo.yaml"
	DefaultMarker           = "<!-- ggo:v1 -->"
	GlobalDirName           = ".ggo"
)

type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	Bot    BotConfig    `yaml:"bot"`
}

type GitHubConfig struct {
	AppID          int64  `yaml:"app_id,omitempty"`
	InstallationID int64  `yaml:"installation_id"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

type BotConfig struct {
	Marker string `yaml:"marker"`
	DryRun bool   `yaml:"dry_run"`
}

func Load(path string) (*Config, string, error) {
	configPath, found, err := ResolvePath(path)
	if err != nil {
		return nil, "", err
	}

	cfg := defaultConfig()
	if found {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, "", fmt.Errorf("read config %q: %w", configPath, err)
		}

		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, "", fmt.Errorf("parse config %q: %w", configPath, err)
		}
	}

	envOverrides, err := applyEnv(&cfg)
	if err != nil {
		return nil, "", err
	}

	if cfg.Bot.Marker == "" {
		cfg.Bot.Marker = DefaultMarker
	}
	cfg.GitHub.AppID = DefaultAppID

	if err := cfg.Validate(); err != nil {
		if !found {
			return nil, "", fmt.Errorf("no config file found; set env vars or create %s in the current directory or ~/.ggo: %w", DefaultConfigName, err)
		}
		return nil, "", err
	}

	if cfg.GitHub.PrivateKeyPath != "" && !filepath.IsAbs(cfg.GitHub.PrivateKeyPath) {
		if found && !envOverrides.PrivateKeyPath {
			cfg.GitHub.PrivateKeyPath = filepath.Join(filepath.Dir(configPath), cfg.GitHub.PrivateKeyPath)
		} else {
			cfg.GitHub.PrivateKeyPath = filepath.Clean(cfg.GitHub.PrivateKeyPath)
		}
	}

	return &cfg, configPath, nil
}

func ResolvePath(explicitPath string) (string, bool, error) {
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err != nil {
			return "", false, fmt.Errorf("config path %q is not readable: %w", explicitPath, err)
		}
		return explicitPath, true, nil
	}

	if envPath := os.Getenv("GGO_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err != nil {
			return "", false, fmt.Errorf("GGO_CONFIG path %q is not readable: %w", envPath, err)
		}
		return envPath, true, nil
	}

	candidates := []string{DefaultConfigName}
	if globalPath, err := GlobalPath(); err == nil {
		candidates = append(candidates, globalPath)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("check config path %q: %w", candidate, err)
		}
	}

	return "", false, nil
}

func GlobalDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	if home == "" {
		return "", errors.New("home directory is empty")
	}
	return filepath.Join(home, GlobalDirName), nil
}

func GlobalPath() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, DefaultConfigName), nil
}

func (c Config) Validate() error {
	if c.GitHub.AppID <= 0 {
		return errors.New("github.app_id must be set to a positive integer")
	}
	if c.GitHub.InstallationID <= 0 {
		return errors.New("github.installation_id must be set to a positive integer")
	}
	if c.GitHub.PrivateKeyPath == "" {
		return errors.New("github.private_key_path must be set")
	}
	if c.Bot.Marker == "" {
		return errors.New("bot.marker must be set")
	}
	return nil
}

func defaultConfig() Config {
	return Config{
		GitHub: GitHubConfig{
			AppID: DefaultAppID,
		},
		Bot: BotConfig{
			Marker: DefaultMarker,
		},
	}
}

type envOverrides struct {
	PrivateKeyPath bool
}

func applyEnv(cfg *Config) (envOverrides, error) {
	var overrides envOverrides

	if value, ok := firstEnv("GGO_INSTALLATION_ID", "GGOBOONG_INSTALLATION_ID"); ok {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return overrides, fmt.Errorf("parse GGO_INSTALLATION_ID/GGOBOONG_INSTALLATION_ID: %w", err)
		}
		cfg.GitHub.InstallationID = parsed
	}
	if value, ok := firstEnv("GGO_PRIVATE_KEY_PATH", "GGOBOONG_PRIVATE_KEY_PATH"); ok {
		cfg.GitHub.PrivateKeyPath = value
		overrides.PrivateKeyPath = true
	}
	if value, ok := firstEnv("GGO_MARKER", "GGOBOONG_MARKER"); ok {
		cfg.Bot.Marker = value
	}
	if value, ok := firstEnv("GGO_DRY_RUN", "GGOBOONG_DRY_RUN"); ok {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return overrides, fmt.Errorf("parse GGO_DRY_RUN/GGOBOONG_DRY_RUN: %w", err)
		}
		cfg.Bot.DryRun = parsed
	}

	return overrides, nil
}

func firstEnv(keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value, true
		}
	}
	return "", false
}
