// This test-only bridge fixture never controls processes. It lets browser E2E
// tests validate the real Native Messaging channel and crash cleanup safely.
package main

import (
	"encoding/json"
	"fmt"
	"github.com/Karench0/focus-lock/internal/bridge"
	"github.com/Karench0/focus-lock/internal/model"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	var mu sync.Mutex
	rules := bridge.Rules{Mode: "strict", Language: "en", LeaseMS: 6000, Websites: []model.Website{}}
	if os.Getenv("FOCUS_LOCK_PIPE_SUFFIX") == "" {
		panic("test pipe suffix required")
	}
	host, err := filepath.Abs(os.Getenv("FOCUS_LOCK_TEST_HOST"))
	if err != nil {
		panic(err)
	}
	server, err := bridge.Start(host, func() bridge.Rules { mu.Lock(); defer mu.Unlock(); return rules }, func(int) {})
	if err != nil {
		panic(err)
	}
	defer server.Close()
	token := os.Getenv("FOCUS_LOCK_TEST_TOKEN")
	if len(token) < 32 {
		panic("test token required")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", 401)
			return
		}
		if r.Method == "POST" {
			var next bridge.Rules
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536)).Decode(&next) != nil {
				http.Error(w, "invalid", 400)
				return
			}
			mu.Lock()
			rules = next
			mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(server.Status())
	})
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, "<!doctype html><title>Focus Lock test destination</title><h1>Allowed test destination</h1>")
	})
	if err = os.WriteFile(os.Getenv("FOCUS_LOCK_TEST_URL_FILE"), []byte("http://"+listener.Addr().String()), 0600); err != nil {
		panic(err)
	}
	srv := http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	if err = srv.Serve(listener); err != nil {
		panic(err)
	}
}
