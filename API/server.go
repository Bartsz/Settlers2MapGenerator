package api

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	domain "settlers2-final-go/DomainModel"
	engine "settlers2-final-go/InternalMapManipulation"
)

// NewHandler serves the embedded UI and generated maps from the same process.
// Files on the user's disk are never accepted as a path from HTTP input.
func NewHandler(static fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		content, err := fs.ReadFile(static, "index.html")
		if err != nil {
			http.Error(w, "UI asset missing", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(content)
	})
	render := func(kind string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			s, err := engine.Generate(data)
			if err != nil {
				http.Error(w, err.Error(), 422)
				return
			}
			switch kind {
			case "preview":
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_, _ = w.Write(s.Preview)
			case "download":
				w.Header().Set("Content-Type", "application/octet-stream")
				size := binary.LittleEndian.Uint16(s.SWD[0x92c:])
				w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Settlers_%dx%d.swd"`, size, size))
				_, _ = w.Write(s.SWD)
			case "save":
				var input domain.SaveAndRunRequest
				_ = json.Unmarshal(data, &input)
				size := binary.LittleEndian.Uint16(s.SWD[0x92c:])
				name := fmt.Sprintf("Settlers_%d_%dp_%dx%d.swd", input.Seed, input.Sym, size, size)
				path, err := saveAndOpen(name, s.SWD)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				_ = json.NewEncoder(w).Encode(map[string]string{"path": path, "status": "saved"})
			}
		}
	}
	mux.HandleFunc("POST /api/map", render("preview"))
	mux.HandleFunc("POST /api/download", render("download")) // preserve existing endpoint
	mux.HandleFunc("POST /api/save-and-run", render("save"))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Private localhost app: reject external Host and cross-origin browser requests.
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if host != "localhost" && host != "127.0.0.1" && host != "[::1]" && host != "::1" {
			http.Error(w, "localhost only", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			prefix := "http://" + r.Host
			if origin != prefix {
				http.Error(w, "cross-origin request denied", 403)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

func saveAndOpen(name string, data []byte) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Downloads")
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	// Never trust client-controlled paths; fixed ASCII pattern only.
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '.' || r == '-') {
			return "", fmt.Errorf("invalid filename")
		}
	}
	if !strings.HasSuffix(name, ".swd") {
		return "", fmt.Errorf("invalid file extension")
	}
	path := filepath.Join(dir, name)
	tmp, err := os.CreateTemp(dir, ".settlers-*.tmp")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	if err = tmp.Close(); err != nil {
		return "", err
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return "", err
	}
	// Do not launch shells with untrusted arguments. OS file association opens SWD.
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return path, fmt.Errorf("saved %s but could not open associated editor: %w", path, err)
	}
	go func() { _ = cmd.Wait() }()
	return path, nil
}
