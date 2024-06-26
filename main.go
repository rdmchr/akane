package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

var (
	port    string
	name    string
	verbose bool
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&port, "port", getEnv("WHOAMI_PORT_NUMBER", "80"), "give me a port number")
	flag.StringVar(&name, "name", os.Getenv("WHOAMI_NAME"), "give me a name")
}

// Data whoami information.
type Data struct {
	IP         []string    `json:"ip,omitempty"`
	Headers    http.Header `json:"headers,omitempty"`
	RemoteAddr string      `json:"remoteAddr,omitempty"`
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.Handle("/api", handle(apiHandler, verbose))
	mux.Handle("/", handle(whoamiHandler, verbose))

	log.Printf("Starting up on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))

}

func handle(next http.HandlerFunc, verbose bool) http.Handler {
	if !verbose {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next(w, r)

		// <remote_IP_address> - [<timestamp>] "<request_method> <request_path> <request_protocol>" -
		log.Printf("%s - - [%s] \"%s %s %s\" - -", r.RemoteAddr, time.Now().Format("02/Jan/2006:15:04:05 -0700"), r.Method, r.URL.Path, r.Proto)
	})
}

func whoamiHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	_, _ = fmt.Fprintln(w, "Hostname:", hostname)

	for _, ip := range getIPs() {
		_, _ = fmt.Fprintln(w, "IP:", ip)
	}

	_, _ = fmt.Fprintln(w, "RemoteAddr:", r.RemoteAddr)
	if err := r.Write(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	data := Data{
		IP:         getIPs(),
		Headers:    r.Header,
		RemoteAddr: r.RemoteAddr,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getIPs() []string {
	var ips []string

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips
}
