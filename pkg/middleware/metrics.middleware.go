package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
)

type MetricsMiddleware struct {
	mu       sync.Mutex
	counters map[string]int
	filePath string
}

func NewMetricsMiddleware(filePath string) *MetricsMiddleware {
	counters := make(map[string]int)
	if _, err := os.Stat(filePath); err == nil {
		file, err := os.Open(filePath)
		if err != nil {
			slog.Error(fmt.Sprintf("Error opening json counter file: %v\n", err))
		}
		defer file.Close()

		err = json.NewDecoder(file).Decode(&counters)
		if err != nil {
			slog.Error(fmt.Sprintf("Error deconing json counter file: %v\n", err))
		}
	}

	return &MetricsMiddleware{
		counters: counters,
		filePath: filePath,
	}
}

func (m *MetricsMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()

		if path == "/posts/{url}" {
			vars := mux.Vars(r)
			id := vars["url"]
			path = "/posts/" + id
		}

		m.mu.Lock()
		m.counters[path]++
		m.mu.Unlock()

		next.ServeHTTP(w, r)

		m.saveCounters()
	})
}

func (m *MetricsMiddleware) saveCounters() {
	m.mu.Lock()
	defer m.mu.Unlock()

	file, err := os.Create(m.filePath)
	if err != nil {
		slog.Error(fmt.Sprintf("Error saving counters: %v\n", err))
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(m.counters)
	if err != nil {
		slog.Error(err.Error())
	}
}
