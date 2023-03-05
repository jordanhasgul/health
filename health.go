package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
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
		healths := make([]*Health, 0, len(checkers))

		var (
			wg     sync.WaitGroup
			status = http.StatusOK
		)
		for name, checker := range checkers {
			wg.Add(1)
			go func(name string, checker Checker) {
				defer wg.Done()

				health := &Health{
					Name:  name,
					State: string(Healthy),
					Time:  time.Now(),
				}

				err := doCheck(checker)
				if err != nil {
					health.State = string(Unhealthy)
					health.Error = err.Error()

					status = http.StatusInternalServerError
				}
				healths = append(healths, health)
			}(name, checker)
		}
		wg.Wait()

		data, _ := json.Marshal(healths)
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		w.Write(data)
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
