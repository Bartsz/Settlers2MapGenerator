package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"settlers2-final-go/API"
	engine "settlers2-final-go/InternalMapManipulation"
)

//go:embed static/index.html static/code.js static/styles.css
var staticFiles embed.FS

func main() {
	if len(os.Args) > 1 {
		if err := engine.ExportFromCLI(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	address := "http://" + ln.Addr().String()
	fmt.Println("Settlers II Generator:", address)
	if runtime.GOOS == "windows" {
		go func() {
			time.Sleep(350 * time.Millisecond)
			_ = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", address).Start()
		}()
	}
	if err = http.Serve(ln, api.NewHandler(files)); err != nil {
		panic(err)
	}
}
