package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const ConfigFile = ".codetranslate.yaml"

type Config struct {
	SourceDir   string `yaml:"source_dir"`
	SourceLang  string `yaml:"source_lang"`
	TargetDir   string `yaml:"target_dir"`
	TargetLang  string `yaml:"target_lang"`
	LedgerDir   string `yaml:"ledger_dir"`
	Model       string `yaml:"model"`
	Conventions string `yaml:"conventions"`
	Concurrency int    `yaml:"concurrency"`
	MaxRetries  int    `yaml:"max_retries"`
}

func DefaultConfig() *Config {
	return &Config{
		LedgerDir:   ".codetranslate/ledger",
		Model:       "haiku",
		Concurrency: 1,
		MaxRetries:  3,
	}
}

func Load() (*Config, error) {
	path, err := FindConfigFile()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

func (c *Config) Save(dir string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	path := filepath.Join(dir, ConfigFile)
	return os.WriteFile(path, data, 0644)
}

func FindConfigFile() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		path := filepath.Join(dir, ConfigFile)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found (run 'translate init' first)", ConfigFile)
		}
		dir = parent
	}
}

func ProjectRoot() (string, error) {
	path, err := FindConfigFile()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}
