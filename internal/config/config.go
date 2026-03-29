package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Duration wraps time.Duration with JSON marshal/unmarshal supporting strings like "10s", "300ms".
type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

// Config holds all runtime configuration for lan-dash.
type Config struct {
	HTTPPort      int      `json:"http_port"`
	SSHPort       int      `json:"ssh_port"`
	ScanInterval  Duration `json:"scan_interval"`
	PingTimeout   Duration `json:"ping_timeout"`
	PortTimeout   Duration `json:"port_timeout"`
	CommonPorts   []int    `json:"common_ports"`
	DBPath        string   `json:"db_path"`
	SweepWorkers  int      `json:"sweep_workers"`
	NicknamesPath string   `json:"nicknames_path"`
}

// Defaults returns a Config populated with sensible defaults.
func Defaults() Config {
	return Config{
		HTTPPort:     3000,
		SSHPort:      2223,
		ScanInterval: Duration{10 * time.Second},
		PingTimeout:  Duration{300 * time.Millisecond},
		PortTimeout:  Duration{250 * time.Millisecond},
		CommonPorts: []int{
			22, 80, 443, 8080, 8443,
			3000, 5000, 9000,
			32400, // Plex
			1883,  // MQTT / Home Assistant
			8123,  // Home Assistant UI
			5353,  // mDNS
		},
		DBPath:        "lan-dash.db",
		SweepWorkers:  64,
		NicknamesPath: "nicknames.json",
	}
}

// Load reads a JSON config file at path and returns the Config.
// If the file does not exist, it returns the defaults without error.
// If the file exists but cannot be parsed, it returns an error.
func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return cfg, nil
}
