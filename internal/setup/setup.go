package setup

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/yaml.v3"

	"github.com/jungju/ggoboong/internal/config"
)

const privateKeyFileName = "github-app.private-key.pem"

type Options struct {
	InstallationID int64
	PrivateKeyPath string
	Marker         string
	DryRun         bool
	Force          bool
	Stdout         io.Writer
}

func Login(opts Options) error {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if err := applyEnvDefaults(&opts); err != nil {
		return err
	}
	if opts.Marker == "" {
		opts.Marker = config.DefaultMarker
	}
	if err := validateOptions(opts); err != nil {
		return err
	}

	keyBytes, err := os.ReadFile(opts.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("read private key %q: %w", opts.PrivateKeyPath, err)
	}
	if _, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes); err != nil {
		return fmt.Errorf("parse private key %q: %w", opts.PrivateKeyPath, err)
	}

	globalDir, err := config.GlobalDir()
	if err != nil {
		return err
	}
	configPath, err := config.GlobalPath()
	if err != nil {
		return err
	}
	keyPath := filepath.Join(globalDir, privateKeyFileName)

	if err := os.MkdirAll(globalDir, 0o700); err != nil {
		return fmt.Errorf("create config directory %q: %w", globalDir, err)
	}
	if err := chmodIfExists(globalDir, 0o700); err != nil {
		return err
	}

	if !opts.Force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("%s already exists; rerun with --force to overwrite", configPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check config file %q: %w", configPath, err)
		}
		if _, err := os.Stat(keyPath); err == nil {
			return fmt.Errorf("%s already exists; rerun with --force to overwrite", keyPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check private key file %q: %w", keyPath, err)
		}
	}

	if err := writeFileAtomic(keyPath, keyBytes, 0o600); err != nil {
		return fmt.Errorf("write private key %q: %w", keyPath, err)
	}

	cfg := config.Config{
		GitHub: config.GitHubConfig{
			InstallationID: opts.InstallationID,
			PrivateKeyPath: "./" + privateKeyFileName,
		},
		Bot: config.BotConfig{
			DryRun: opts.DryRun,
		},
	}

	configBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("render config yaml: %w", err)
	}
	if err := writeFileAtomic(configPath, configBytes, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", configPath, err)
	}

	fmt.Fprintf(opts.Stdout, "installed config: %s\n", configPath)
	fmt.Fprintf(opts.Stdout, "installed private key: %s\n", keyPath)
	return nil
}

func validateOptions(opts Options) error {
	if opts.InstallationID <= 0 {
		return errors.New("--installation-id must be set to a positive integer")
	}
	if opts.PrivateKeyPath == "" {
		return errors.New("--private-key is required")
	}
	return nil
}

func applyEnvDefaults(opts *Options) error {
	if opts.InstallationID == 0 {
		value, ok := firstEnv("GGO_INSTALLATION_ID", "GGOBOONG_INSTALLATION_ID")
		if ok {
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return fmt.Errorf("parse GGO_INSTALLATION_ID/GGOBOONG_INSTALLATION_ID: %w", err)
			}
			opts.InstallationID = parsed
		}
	}
	if opts.PrivateKeyPath == "" {
		if value, ok := firstEnv("GGO_PRIVATE_KEY_PATH", "GGOBOONG_PRIVATE_KEY_PATH"); ok {
			opts.PrivateKeyPath = value
		}
	}
	if opts.Marker == "" {
		if value, ok := firstEnv("GGO_MARKER", "GGOBOONG_MARKER"); ok {
			opts.Marker = value
		}
	}
	if !opts.DryRun {
		if value, ok := firstEnv("GGO_DRY_RUN", "GGOBOONG_DRY_RUN"); ok {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("parse GGO_DRY_RUN/GGOBOONG_DRY_RUN: %w", err)
			}
			opts.DryRun = parsed
		}
	}
	return nil
}

func firstEnv(keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			return value, true
		}
	}
	return "", false
}

func chmodIfExists(path string, mode os.FileMode) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("check %q: %w", path, err)
	}
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("chmod %q: %w", path, err)
	}
	return nil
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}
