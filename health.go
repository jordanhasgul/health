package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// State represents the state of a service dependency.
type State string

const (
	Healthy   State = "healthy"   // A healthy service dependency.
	Unhealthy State = "unhealthy" // An unhealthy service dependency.
)

// Checker performs a health check on a service dependency. Calls to
// Check() must be safe for concurrent access via multiple goroutines.
type Checker interface {
	Check() error
}

// CheckerFunc is an adapter, which is itself a Checker, that allows
// the use of ordinary functions to perform a check.
type CheckerFunc func() error

// Check calls f.
func (f CheckerFunc) Check() error {
	return f()
}

// Health represents the health of a service dependency.
type Health struct {
	Name  string `json:"name"`
	State string `json:"state"`

	Time time.Time `json:"time"`

	Error string `json:"error,omitempty"`
}

// Handler returns an http.Handler that handles health check requests.
func Handler(checkers map[string]Checker) http.Handler {
	f := func(w http.ResponseWriter, req *http.Request) {
		var (
			status   = http.StatusOK
			healthCh = make(chan Health, len(checkers))
		)
		for name, checker := range checkers {
			go func(name string, checker Checker) {
				health := Health{
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
				healthCh <- health
			}(name, checker)
		}

		healths := make([]Health, 0, len(checkers))
		for len(healths) != cap(healths) {
			healths = append(healths, <-healthCh)
		}
		close(healthCh)

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
		3*time.Second,
	)
	defer cancel()

	errs := make(chan error)
	go func() {
		defer func() {
			recv := recover()
			if recv != nil {
				switch t := recv.(type) {
				case string, error, fmt.Stringer:
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
			errs <- ctx.Err()
		}
	}()

	return <-errs
}
