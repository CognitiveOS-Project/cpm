package config

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"
)

type Config struct {
	DefaultRegistry string
}

func Load(path string) (*Config, error) {
	cfg := &Config{DefaultRegistry: "https://registry.cognitive-os.org/v1"}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	file, err := ini.Load(data)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	sec := file.Section("default")
	if sec != nil && sec.HasKey("url") {
		cfg.DefaultRegistry = sec.Key("url").String()
	}
	return cfg, nil
}

func RegistriesPath() string {
	return "/etc/cognitiveos/registries.toml"
}
