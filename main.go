package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type servicesServer struct {
	addr  string
	proxy *httputil.ReverseProxy
}

func newServer(addr string) *servicesServer {
	serverURL, err := url.Parse(addr)
	handleError(err)

	return &servicesServer{
		addr:  addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}
}

type LoadBalancer struct {
	port    string
	servers map[string]Server // Use a map for easy path-to-server mapping
}

func NewLoadBalancer(port string, servers map[string]Server) *LoadBalancer {
	return &LoadBalancer{
		port:    port,
		servers: servers,
	}
}

func handleError(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func (s *servicesServer) Address() string {
	return s.addr
}

func (s *servicesServer) IsAlive() bool {
	return true
}

func (s *servicesServer) Serve(rw http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(rw, r)
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if server, ok := lb.servers[path]; ok {
		fmt.Printf("forwarding request to address %q\n", server.Address())
		server.Serve(rw, r)
	} else {
		http.Error(rw, "Not Found", http.StatusNotFound)
	}
}

func main() {
	servers := map[string]Server{
		"/trip":         newServer("http://localhost:3081"), // Trip service
		"/notification": newServer("http://localhost:3082"), // Notification service
		"/geolocation":  newServer("http://localhost:3083"), // Geolocation service
	}
	lb := NewLoadBalancer("9090", servers)

	handleRedirect := func(rw http.ResponseWriter, r *http.Request) {
		lb.serveProxy(rw, r)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("Serving requests at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
