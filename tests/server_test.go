package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"settlers2-final-go/API"
)

func TestAPIAndEmbeddedStaticContracts(t *testing.T) {
	h := api.NewHandler(os.DirFS("../static"))
	for _, target := range []string{"/", "/static/code.js", "/static/styles.css"} {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080"+target, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Fatalf("GET %s => %d", target, w.Code)
		}
	}
	req := parameters(3)
	previewReq := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/map", bytes.NewReader(req))
	previewW := httptest.NewRecorder()
	h.ServeHTTP(previewW, previewReq)
	if previewW.Code != 200 {
		t.Fatalf("preview %d: %s", previewW.Code, previewW.Body.String())
	}
	var p map[string]any
	if err := json.Unmarshal(previewW.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	binaryReq := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/download", bytes.NewReader(req))
	binaryW := httptest.NewRecorder()
	h.ServeHTTP(binaryW, binaryReq)
	if binaryW.Code != 200 || !bytes.HasPrefix(binaryW.Body.Bytes(), []byte("WORLD_V1.0")) {
		t.Fatalf("SWD download %d", binaryW.Code)
	}
	forbidden := httptest.NewRequest(http.MethodPost, "http://example.com:8080/api/map", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, forbidden)
	if w.Code != 403 {
		t.Fatal("external host not rejected")
	}
	invalid := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/download", strings.NewReader(`{"sym":99}`))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, invalid)
	if w.Code != 422 || w.Body.Len() == 317577 {
		t.Fatal("bad configuration downloaded incomplete SWD")
	}
}

// Save and Run must write inside the current user's Downloads directory and
// invoke the configured file association (stubbed here on Linux).
func TestSaveAndRunWritesDownloadsAndOpensAssociatedApp(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("association command is stubbed on Linux")
	}
	root := t.TempDir()
	t.Setenv("HOME", root)
	bindir := filepath.Join(root, "bin")
	if err := os.MkdirAll(bindir, 0755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(root, "opened.txt")
	stub := filepath.Join(bindir, "xdg-open")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s' \"$1\" > "+marker+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bindir+string(os.PathListSeparator)+os.Getenv("PATH"))
	h := api.NewHandler(os.DirFS("../static"))
	payload := parameters(2)
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:9000/api/save-and-run", bytes.NewReader(payload))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var answer struct {
		Path   string `json:"path"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &answer); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "Downloads", "Settlers_123456_2p_108x108.swd")
	if answer.Path != want || answer.Status != "saved" {
		t.Fatalf("wrong result %+v", answer)
	}
	binary, err := os.ReadFile(answer.Path)
	if err != nil || !bytes.HasPrefix(binary, []byte("WORLD_V1.0")) {
		t.Fatalf("download was not written: %v", err)
	}
	var invoked []byte
	for i := 0; i < 100; i++ {
		invoked, err = os.ReadFile(marker)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || string(invoked) != want {
		t.Fatalf("file association was not invoked correctly: %q, %v", invoked, err)
	}
}
