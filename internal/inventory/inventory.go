package inventory

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Inventory struct {
	Targets []Target `yaml:"targets"`
}

type Target struct {
	Name    string            `yaml:"name"`
	Address string            `yaml:"address"` // host or ip (no port here by default)
	Mode    string            `yaml:"mode"`    // "ssh" (we keep room for "snmp" later)
	Labels  map[string]string `yaml:"labels"`
}

// NormalizedTarget is what the rest of the system consumes (validated + defaults applied).
type NormalizedTarget struct {
	Name        string
	Address     string // host/ip
	Mode        string // ssh
	Labels      map[string]string
	DisplayName string // usually same as Name (handy for UI/metrics labels)
}

var (
	nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)
)

// Load reads and validates targets.yaml.
func Load(path string) (*Inventory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}

	var inv Inventory
	if err := yaml.Unmarshal(b, &inv); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if len(inv.Targets) == 0 {
		return nil, errors.New("inventory has no targets")
	}
	return &inv, nil
}

// Normalize validates and applies defaults. This is what other packages should call.
func (inv *Inventory) Normalize() ([]NormalizedTarget, error) {
	seen := map[string]bool{}
	out := make([]NormalizedTarget, 0, len(inv.Targets))

	for i, t := range inv.Targets {
		idx := i + 1

		t.Name = strings.TrimSpace(t.Name)
		t.Address = strings.TrimSpace(t.Address)
		t.Mode = strings.TrimSpace(strings.ToLower(t.Mode))

		if t.Name == "" {
			return nil, fmt.Errorf("targets[%d]: name is required", idx)
		}
		if !nameRe.MatchString(t.Name) {
			return nil, fmt.Errorf("targets[%d]: invalid name %q (allowed: letters/digits . _ -)", idx, t.Name)
		}
		if seen[t.Name] {
			return nil, fmt.Errorf("targets[%d]: duplicate name %q", idx, t.Name)
		}
		seen[t.Name] = true

		if t.Address == "" {
			return nil, fmt.Errorf("targets[%d] (%s): address is required", idx, t.Name)
		}
		if strings.Contains(t.Address, "://") {
			return nil, fmt.Errorf("targets[%d] (%s): address must be host/ip only (no scheme): got %q", idx, t.Name, t.Address)
		}
		if strings.Contains(t.Address, "/") {
			return nil, fmt.Errorf("targets[%d] (%s): address must not contain path: got %q", idx, t.Name, t.Address)
		}

		if t.Mode == "" {
			t.Mode = "ssh" // default
		}
		if t.Mode != "ssh" {
			return nil, fmt.Errorf("targets[%d] (%s): unsupported mode %q (only ssh for now)", idx, t.Name, t.Mode)
		}

		labels := map[string]string{}
		for k, v := range t.Labels {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" || v == "" {
				continue
			}
			labels[k] = v
		}

		// Always enforce stable labels
		labels["target"] = t.Name
		labels["mode"] = t.Mode

		out = append(out, NormalizedTarget{
			Name:        t.Name,
			Address:     t.Address,
			Mode:        t.Mode,
			Labels:      labels,
			DisplayName: t.Name,
		})
	}

	// deterministic order (helps diff + tests)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out, nil
}

