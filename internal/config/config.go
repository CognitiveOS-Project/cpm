package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type RegistryEntry struct {
	Name string
	URL  string
}

type Registries struct {
	Official     OfficialRegistries
	Alternatives map[string]string
}

type OfficialRegistries struct {
	Primary string
	Mirrors map[string]string
}

func Load(path string) (*Registries, error) {
	r := defaultRegistries()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var file struct {
		Official     struct {
			Primary string            `toml:"primary"`
			Mirrors map[string]string `toml:"mirrors"`
		} `toml:"official"`
		Alternatives map[string]string `toml:"alternative"`
	}
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if file.Official.Primary != "" {
		r.Official.Primary = file.Official.Primary
	}
	if file.Official.Mirrors != nil {
		r.Official.Mirrors = file.Official.Mirrors
	}
	if file.Alternatives != nil {
		r.Alternatives = file.Alternatives
	}

	return r, nil
}

func defaultRegistries() *Registries {
	return &Registries{
		Official: OfficialRegistries{
			Primary: "https://registry-us-all-distros-official.cognitive-os.org/v1",
			Mirrors: map[string]string{},
		},
		Alternatives: map[string]string{},
	}
}

func RegistriesPath() string {
	return "/etc/cognitiveos/registries.toml"
}

func (r *Registries) Resolve(section string) (string, error) {
	if section == "" {
		return r.Official.Primary, nil
	}

	parts := strings.SplitN(section, ".", 2)

	switch parts[0] {
	case "official":
		if len(parts) == 1 {
			return r.Official.Primary, nil
		}
		name := parts[1]
		if url, ok := r.Official.Mirrors[name]; ok {
			return url, nil
		}
		return "", fmt.Errorf("official mirror %q not found in registries", name)

	case "alternative":
		if len(parts) < 2 {
			return "", fmt.Errorf("alternative section requires a name, e.g. alternative.community")
		}
		url, ok := r.Alternatives[parts[1]]
		if !ok {
			return "", fmt.Errorf("alternative registry %q not found in registries", parts[1])
		}
		return url, nil

	default:
		return "", fmt.Errorf("unknown registry section %q (expected official or alternative)", parts[0])
	}
}

func (r *Registries) All() []RegistryEntry {
	var entries []RegistryEntry

	if r.Official.Primary != "" {
		entries = append(entries, RegistryEntry{Name: "official", URL: r.Official.Primary})
	}

	for name, url := range r.Official.Mirrors {
		entries = append(entries, RegistryEntry{Name: "official." + name, URL: url})
	}

	for name, url := range r.Alternatives {
		entries = append(entries, RegistryEntry{Name: "alternative." + name, URL: url})
	}

	return entries
}
