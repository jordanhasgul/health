package health

import (
	"context"
	"errors"
	"fmt"
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
