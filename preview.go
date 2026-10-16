//go:build windows

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"time"
)

// preview is an explicit diagnostic mode with real storage and timers and no
// process enforcement. Its loopback API requires a per-run random bearer token.
func (a *App) preview() error {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	token := hex.EncodeToString(b)
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := l.Addr().String()
	origin := "http://" + addr
	mux := http.NewServeMux()
	mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Host != addr || r.Header.Get("Authorization") != "Bearer "+token || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origin) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
		var request struct {
			Command string          `json:"command"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "invalid request", 400)
			return
		}
		if request.Command == "registerBridge" || request.Command == "openExtension" || request.Command == "quit" || request.Command == "tray" || request.Command == "pickApplication" {
			http.Error(w, "requires desktop app", 400)
			return
		}
		out, err := a.Call(request.Command, string(request.Payload))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(out))
	})
	root, err := fs.Sub(assets, "ui/dist")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(root)))
	url := origin + "/#token=" + token
	// Tests can use an output file when the executable is linked as windowsgui.
	if path := os.Getenv("FOCUS_LOCK_PREVIEW_URL_FILE"); path != "" {
		if err = os.WriteFile(path, []byte(url), 0600); err != nil {
			return err
		}
	}
	fmt.Println(url)
	server := http.Server{Handler: mux, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 20 * time.Second}
	return server.Serve(l)
}
