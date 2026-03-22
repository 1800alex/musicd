package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds persistent user settings.
type Config struct {
	ServerURL string `json:"server_url"`
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "musictui", "config.json")
}

// LoadConfig reads the config file, returning defaults if it doesn't exist.
func LoadConfig() *Config {
	cfg := &Config{
		ServerURL: "http://localhost:8080",
	}
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, cfg)
	return cfg
}

// SaveConfig writes the config file to disk.
func SaveConfig(cfg *Config) error {
	p := configPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
