package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"gpsdr.local/gpsdr/web"
)

func main() {
	defaultData := defaultDataDirectory()
	host := flag.String("listen", "127.0.0.1", "address to listen on")
	port := flag.Int("port", 8073, "web console port")
	data := flag.String("data", defaultData, "data directory")
	demo := flag.Bool("demo", false, "enable clearly marked simulated receiver activity")
	token := flag.String("token", "", "access token required by non-local clients")
	openBrowser := flag.Bool("open", false, "open the web console in the default browser")
	repairMedia := flag.Bool("repair-media", false, "reconcile saved recording references without starting receivers; stop the app first")
	passwordHashStdin := flag.Bool("password-hash-stdin", false, "read one password from standard input and print a password hash")
	flag.Parse()
	if *passwordHashStdin {
		password, err := io.ReadAll(io.LimitReader(os.Stdin, 1025))
		if err != nil {
			log.Fatal(err)
		}
		hash, err := app.NewPasswordHash(strings.TrimSpace(string(password)))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(hash)
		return
	}
	// Service managers keep credentials in an EnvironmentFile. Prefer that
	// path to placing the token in ExecStart, where it is visible in process
	// listings and service diagnostics.
	if strings.TrimSpace(*token) == "" {
		*token = strings.TrimSpace(os.Getenv("GPSDR_WEB_TOKEN"))
	}
	if *repairMedia {
		store, err := app.NewEventStore(filepath.Join(*data, "Data"))
		if err != nil {
			log.Fatal(err)
		}
		report, err := store.ReconcileMedia(*data)
		if err != nil {
			log.Fatal(err)
		}
		_ = json.NewEncoder(os.Stdout).Encode(report)
		return
	}
	if *port < 1 || *port > 65535 {
		log.Fatal("port must be between 1 and 65535")
	}
	if !isLoopback(*host) && *token == "" {
		*token = randomToken()
		log.Printf("Network access token: %s", *token)
	}
	addressHost := displayHost(*host)
	// The runtime/status API must never echo the access token. Authenticated
	// reverse proxies inject it as a header; direct clients may supply it
	// explicitly without having it copied into logs, status, or history.
	url := fmt.Sprintf("http://%s:%d/", addressHost, *port)
	// Serve the touchscreen shell before rebuilding persisted state. On a Pi
	// with a sizeable capture history that restoration can take over a minute.
	server := app.NewServer(nil, web.Files, *host, uint16(*port), *token)
	if err := server.SetPasswordAuthentication(strings.TrimSpace(os.Getenv("GPSDR_WEB_PASSWORD_HASH")), os.Getenv("GPSDR_TRUST_PROXY_HEADERS") == "1"); err != nil {
		log.Fatal(err)
	}
	log.Printf("GP-SDR %s", app.Version)
	log.Printf("Web console: %s", url)
	go func() {
		time.Sleep(300 * time.Millisecond)
		if *openBrowser {
			_ = openURL(url)
		}
	}()
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	runtimeState, err := app.NewRuntime(*data, url, *demo)
	if err != nil {
		log.Fatal(err)
	}
	server.SetRuntime(runtimeState)
	log.Printf("GP-SDR runtime ready")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	runtimeState.Stop()
	app.ShutdownCaptureSessions()
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
		if interfaces, err := net.Interfaces(); err == nil {
			for _, iface := range interfaces {
				if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
					continue
				}
				addresses, _ := iface.Addrs()
				for _, address := range addresses {
					ip, _, err := net.ParseCIDR(address.String())
					if err == nil && ip.To4() != nil && ip.IsPrivate() {
						return ip.String()
					}
				}
			}
		}
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
