package inventory

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Inventory struct {
	Targets []Target `yaml:"targets"`
}

type Target struct {
	Name    string            `yaml:"name"`
	Address string            `yaml:"address"`
	Mode    string            `yaml:"mode"`
	Labels  map[string]string `yaml:"labels"`
	SSH     SSHConfig         `yaml:"ssh"`
}

type SSHConfig struct {
	User string     `yaml:"user"`
	Auth AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Mode        string `yaml:"mode"`         // "password_env" (ตอนนี้ใช้ตัวนี้)
	PasswordEnv string `yaml:"password_env"` // e.g. SSH_PASS_ECS1
	// เผื่ออนาคตจะทำ key-based:
	KeyPath string `yaml:"key_path"`
}

func Load(path string) (*Inventory, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read inventory: %w", err)
	}

	var inv Inventory
	if err := yaml.Unmarshal(b, &inv); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	// normalize defaults
	for i := range inv.Targets {
		t := &inv.Targets[i]
		if t.Mode == "" {
			t.Mode = "ssh"
		}
		if t.Labels == nil {
			t.Labels = map[string]string{}
		}
		if t.SSH.User == "" {
			t.SSH.User = "root" // B-mode: internal/lab
		}
		if t.SSH.Auth.Mode == "" {
			t.SSH.Auth.Mode = "password_env"
		}
	}

	return &inv, nil
}
