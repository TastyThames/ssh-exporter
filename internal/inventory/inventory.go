package inventory

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Inventory struct {
	Targets []Target
}

type Target struct {
	Name    string
	Address string
	Mode    string
	Labels  map[string]string

	SSH SSHConfig
}

type SSHConfig struct {
	User string
	Auth SSHAuth
}

type SSHAuth struct {
	Mode         string // "password_env" | "password_file"
	PasswordEnv  string // e.g. SSH_PASS_ECS1
	PasswordFile string // e.g. /run/secrets/ecs-1.pass
	KeyPath      string // future
}

type rawInventory struct {
	Targets []rawTarget `yaml:"targets"`
}

type rawTarget struct {
	Name    string            `yaml:"name"`
	Address string            `yaml:"address"`
	Mode    string            `yaml:"mode"`
	Labels  map[string]string `yaml:"labels"`

	SSH rawSSH `yaml:"ssh"`
}

type rawSSH struct {
	User string  `yaml:"user"`
	Auth rawAuth `yaml:"auth"`
}

type rawAuth struct {
	Mode         string `yaml:"mode"`
	PasswordEnv  string `yaml:"password_env"`
	PasswordFile string `yaml:"password_file"`
	KeyPath      string `yaml:"key_path"`
}

func Load(path string) (*Inventory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}

	var ri rawInventory
	if err := yaml.Unmarshal(b, &ri); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	out := &Inventory{Targets: make([]Target, 0, len(ri.Targets))}
	for _, t := range ri.Targets {
		name := strings.TrimSpace(t.Name)
		addr := strings.TrimSpace(t.Address)

		if addr == "" {
			return nil, fmt.Errorf("target %q: address is empty", name)
		}
		if name == "" {
			name = addr
		}

		mode := strings.TrimSpace(t.Mode)
		if mode == "" {
			mode = "ssh"
		}

		labels := t.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		labels["name"] = name

		// Defaults for SSH
		user := strings.TrimSpace(t.SSH.User)
		if user == "" {
			user = "root" // lab-friendly default
		}

		authMode := strings.TrimSpace(t.SSH.Auth.Mode)
		passEnv := strings.TrimSpace(t.SSH.Auth.PasswordEnv)
		passFile := strings.TrimSpace(t.SSH.Auth.PasswordFile)
		keyPath := strings.TrimSpace(t.SSH.Auth.KeyPath)

		// ---- Smart default for auth mode ----
		if authMode == "" {
			if passFile != "" {
				authMode = "password_file"
			} else {
				authMode = "password_env"
			}
		}

		// Validate
		switch mode {
		case "ssh":
			switch authMode {
			case "password_env":
				if passEnv == "" {
					return nil, fmt.Errorf("target %q: ssh.auth.password_env is required for password_env mode", name)
				}
			case "password_file":
				if passFile == "" {
					return nil, fmt.Errorf("target %q: ssh.auth.password_file is required for password_file mode", name)
				}
			default:
				return nil, fmt.Errorf("target %q: unsupported ssh.auth.mode %q", name, authMode)
			}
		default:
			return nil, fmt.Errorf("target %q: unsupported mode %q", name, mode)
		}

		out.Targets = append(out.Targets, Target{
			Name:    name,
			Address: addr,
			Mode:    mode,
			Labels:  labels,
			SSH: SSHConfig{
				User: user,
				Auth: SSHAuth{
					Mode:         authMode,
					PasswordEnv:  passEnv,
					PasswordFile: passFile,
					KeyPath:      keyPath,
				},
			},
		})
	}

	return out, nil
}
