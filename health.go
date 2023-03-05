package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type State string

const (
	Healthy   State = "healthy"
	Unhealthy State = "unhealthy"
)

type Checker interface {
	Check() error
}

type CheckerFunc func() error

func (f CheckerFunc) Check() error {
	return f()
}

type Health struct {
	Name  string `json:"name"`
	State string `json:"state"`

	Time time.Time `json:"time"`

	Error string `json:"error,omitempty"`
}

func Handler(checkers map[string]Checker) http.Handler {
	f := func(w http.ResponseWriter, r *http.Request) {

	}
	return http.HandlerFunc(f)
}

func doCheck(checker Checker) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	errs := make(chan error)
	go func() {
		defer func() {
			recv := recover()
			if recv != nil {
				switch t := recv.(type) {
				case fmt.Stringer:
					errs <- fmt.Errorf("%s", t)
				default:
					panic(t)
				}
			}

			close(errs)
		}()

		select {
		case errs <- checker.Check():
		case <-ctx.Done():
			errs <- errors.New("timeout exceeded")
		}
	}()

	return <-errs
}
