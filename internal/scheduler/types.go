package scheduler

type Job struct {
	Target string
	Labels map[string]string
}
