package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultLineWidth   = 120
	defaultAlarmWindow = time.Hour
)

type Config struct {
	General struct {
		LineWidth   int           `yaml:"line_width"`
		AlarmWindow time.Duration `yaml:"-"`
	} `yaml:"general"`
}

type rawConfig struct {
	General struct {
		LineWidth   int    `yaml:"line_width"`
		AlarmWindow string `yaml:"alarm_window"`
	} `yaml:"general"`
}

var (
	cfgOnce sync.Once
	cfg     *Config
)

func configPath() string {
	if p := os.Getenv("TODO_PATH"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "todo", ".todo")
}

func getConfig() *Config {
	cfgOnce.Do(func() {
		cfg = loadConfig()
	})
	return cfg
}

func loadConfig() *Config {
	c := &Config{}
	c.General.LineWidth = defaultLineWidth
	c.General.AlarmWindow = defaultAlarmWindow

	path := configPath()
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c // missing/unreadable config → defaults
	}
	var parsed rawConfig
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid config at %s: %v\n", path, err)
		return c
	}
	if parsed.General.LineWidth > 0 {
		c.General.LineWidth = parsed.General.LineWidth
	}
	if s := strings.TrimSpace(parsed.General.AlarmWindow); s != "" {
		if d, err := parseWindow(s); err == nil {
			c.General.AlarmWindow = d
		} else {
			fmt.Fprintf(os.Stderr, "warning: invalid alarm_window %q in %s: %v\n", s, path, err)
		}
	}
	return c
}

func parseWindow(s string) (time.Duration, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	if d, ok := parseRelative(strings.ToLower(s)); ok {
		return d, nil
	}
	return 0, fmt.Errorf("unrecognized duration %q (try forms like 1h, 30m, 1h30m, 2d, 1w)", s)
}
