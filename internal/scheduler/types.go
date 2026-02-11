package scheduler

type Job struct {
	Target string
	Labels map[string]string

	SSHUser string

	AuthMode     string // "password_env" | "password_file"
	PasswordEnv  string // e.g. SSH_PASS_ECS1
	PasswordFile string // e.g. /run/secrets/ecs-1.pass

	KeyPath string // future
}
