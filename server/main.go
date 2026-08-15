package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gpsdr.local/gpsdr/internal/app"
)

//go:embed web/*
var webFiles embed.FS

func main() {
	defaultData := defaultDataDirectory()
	host := flag.String("listen", "127.0.0.1", "address to listen on")
	port := flag.Int("port", 8073, "web console port")
	data := flag.String("data", defaultData, "data directory")
	demo := flag.Bool("demo", false, "enable clearly marked simulated receiver activity")
	token := flag.String("token", "", "access token required by non-local clients")
	openBrowser := flag.Bool("open", false, "open the web console in the default browser")
	flag.Parse()
	if *port < 1 || *port > 65535 {
		log.Fatal("port must be between 1 and 65535")
	}
	if !isLoopback(*host) && *token == "" {
		*token = randomToken()
		log.Printf("Network access token: %s", *token)
	}
	addressHost := displayHost(*host)
	url := fmt.Sprintf("http://%s:%d/", addressHost, *port)
	if *token != "" {
		url += "?token=" + *token
	}
	runtimeState, err := app.NewRuntime(*data, url, *demo)
	if err != nil {
		log.Fatal(err)
	}
	web, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	server := app.NewServer(runtimeState, web, *host, uint16(*port), *token)
	go func() {
		time.Sleep(300 * time.Millisecond)
		if *openBrowser {
			_ = openURL(url)
		}
	}()
	go func() {
		log.Printf("GP-SDR %s", app.Version)
		log.Printf("Web console: %s", url)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	runtimeState.Stop()
	_ = server.Close()
}

func defaultDataDirectory() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "GP-SDR")
	}
	return filepath.Join(".", "GP-SDR-Data")
}
func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
func displayHost(host string) string {
	if host == "0.0.0.0" || host == "::" || host == "[::]" {
		if name, err := os.Hostname(); err == nil && name != "" {
			return name
		}
		return "localhost"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return "[" + host + "]"
	}
	return host
}
func randomToken() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
func openURL(url string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		args = []string{url}
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		command = "xdg-open"
		args = []string{url}
	}
	return exec.Command(command, args...).Start()
}
