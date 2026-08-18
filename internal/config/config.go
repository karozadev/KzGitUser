// Package config manages KzGitUser's own configuration file, which stores
// named Git identity profiles and (optionally) validation rules used by
// `kzgit check`.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// ErrProfileNotFound is returned when looking up a profile that has not
// been defined.
var ErrProfileNotFound = errors.New("profile not found")

// ErrProfileExists is returned by Add when a profile with the same name
// already exists.
var ErrProfileExists = errors.New("profile already exists")

// ErrInvalidEmail is returned when a profile is given a malformed email
// address.
var ErrInvalidEmail = errors.New("invalid email address")

// ErrInvalidName is returned when a profile is given an empty name or an
// empty display name.
var ErrInvalidName = errors.New("invalid name")

// Profile is a saved Git identity that can be applied to a repository.
type Profile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Rules holds the validation rules enforced by `kzgit check`.
type Rules struct {
	// AllowedDomains, when non-empty, restricts the local identity's
	// email to one of these domains (e.g. "company.com").
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

// Config is the persisted KzGitUser configuration.
type Config struct {
	Profiles map[string]Profile `json:"profiles"`
	Rules    Rules              `json:"rules,omitempty"`

	path string
}

var emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// ValidateEmail reports whether email looks like a syntactically valid
// email address.
func ValidateEmail(email string) bool {
	return emailRE.MatchString(email)
}

// DefaultPath returns the primary config file location:
// $XDG_CONFIG_HOME/kzgit/config.json, defaulting to ~/.config/kzgit/config.json.
func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "kzgit", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "kzgit", "config.json"), nil
}

// FallbackPath returns the legacy single-file location: ~/.kzgit.json.
func FallbackPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kzgit.json"), nil
}

// Load reads the KzGitUser configuration from its default location,
// falling back to the legacy ~/.kzgit.json path if the default one does not
// exist. It returns an empty, ready-to-use Config if neither file exists.
func Load() (*Config, error) {
	primary, err := DefaultPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(primary); err == nil {
		return LoadFrom(primary)
	}

	fallback, err := FallbackPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(fallback); err == nil {
		return LoadFrom(fallback)
	}

	return &Config{Profiles: map[string]Profile{}, path: primary}, nil
}

// LoadFrom reads the KzGitUser configuration from a specific file path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Profiles: map[string]Profile{}, path: path}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	cfg.path = path
	return &cfg, nil
}

// Path returns the file path this Config was loaded from (or will be saved
// to).
func (c *Config) Path() string {
	return c.path
}

// Save writes the configuration back to disk, creating parent directories
// as needed.
func (c *Config) Save() error {
	if c.path == "" {
		p, err := DefaultPath()
		if err != nil {
			return err
		}
		c.path = p
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(c.path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// Add registers a new profile. It fails if the name is already taken or if
// the name/email are invalid.
func (c *Config) Add(name, displayName, email string) error {
	if name == "" {
		return fmt.Errorf("%w: profile name cannot be empty", ErrInvalidName)
	}
	if displayName == "" {
		return fmt.Errorf("%w: display name cannot be empty", ErrInvalidName)
	}
	if !ValidateEmail(email) {
		return fmt.Errorf("%w: %q", ErrInvalidEmail, email)
	}
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	if _, exists := c.Profiles[name]; exists {
		return fmt.Errorf("%w: %q", ErrProfileExists, name)
	}
	c.Profiles[name] = Profile{Name: displayName, Email: email}
	return nil
}

// Update overwrites an existing profile's display name and/or email.
func (c *Config) Update(name, displayName, email string) error {
	p, exists := c.Profiles[name]
	if !exists {
		return fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}
	if displayName != "" {
		p.Name = displayName
	}
	if email != "" {
		if !ValidateEmail(email) {
			return fmt.Errorf("%w: %q", ErrInvalidEmail, email)
		}
		p.Email = email
	}
	c.Profiles[name] = p
	return nil
}

// Remove deletes a profile by name.
func (c *Config) Remove(name string) error {
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}
	delete(c.Profiles, name)
	return nil
}

// Get returns a single profile by name.
func (c *Config) Get(name string) (Profile, error) {
	p, exists := c.Profiles[name]
	if !exists {
		return Profile{}, fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}
	return p, nil
}

// List returns all profiles sorted alphabetically by name.
func (c *Config) List() []NamedProfile {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]NamedProfile, 0, len(names))
	for _, name := range names {
		result = append(result, NamedProfile{Name: name, Profile: c.Profiles[name]})
	}
	return result
}

// NamedProfile pairs a profile with the key it is stored under.
type NamedProfile struct {
	Name string
	Profile
}
