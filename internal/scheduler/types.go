package scheduler

type Job struct {
	Target string
	Labels map[string]string

	SSHUser     string
	AuthMode    string // "password_env"
	PasswordEnv string // e.g. SSH_PASS_ECS1
	KeyPath     string // future
}
