package health

type Health string

const (
	Healthy   Health = "healthy"
	Unhealthy Health = "unhealthy"
)

type Checker interface {
	Check() error
}

type CheckerFunc func() error

func (f CheckerFunc) Check() error {
	return f()
}
