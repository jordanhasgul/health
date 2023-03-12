package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	t.Run("without-checks", func(t *testing.T) {
		checkers := make(map[string]Checker)

		mux := http.NewServeMux()
		mux.Handle("/health", Handler(checkers))

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("with-checks", func(t *testing.T) {
		checkers := make(map[string]Checker)
		checkers["check1"] = CheckerFunc(func() error {
			return nil
		})
		checkers["check2"] = CheckerFunc(func() error {
			return errors.New("error")
		})
		checkers["check3"] = CheckerFunc(func() error {
			panic("panic")
		})
		checkers["check4"] = CheckerFunc(func() error {
			<-time.After(5 * time.Second)
			return nil
		})

		mux := http.NewServeMux()
		mux.Handle("/health", Handler(checkers))

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.Nil(t, err)

		var healths []Health
		err = json.Unmarshal(body, &healths)
		require.Nil(t, err)

		errorMsgs := map[string]string{
			"check1": "",
			"check2": "error",
			"check3": "panic",
			"check4": "context deadline exceeded",
		}
		for _, health := range healths {
			fmt.Println(health)
			errorMsg := errorMsgs[health.Name]
			require.Equal(t, errorMsg, health.Error)
		}

	})
}
