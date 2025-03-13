package server

import (
	"net/http"
	"time"

	"github.com/Zireael26/broken-links-checker/api"
	"github.com/Zireael26/broken-links-checker/internal/scanner"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

type Server struct {
    httpServer *http.Server
    scanner    *scanner.Scanner
    browser    *rod.Browser
}

func New() (*Server, error) {
    // Initialize Chrome instance
    l := launcher.New().
        Headless(true).
        NoSandbox(true).
        Set("disable-gpu").
        Set("disable-extensions").
        Set("blink-settings", "imagesEnabled=false")
    browser := rod.New().ControlURL(l.MustLaunch()).MustConnect()

    // HTTP client for static fetching
    client := &http.Client{Timeout: 10 * time.Second}

    // Scanner instance
    scanner := scanner.New(client, browser)

    // API handler
    handler := api.NewHandler(scanner)

    // Router setup
    router := api.NewRouter(handler)

    // HTTP server
    httpServer := &http.Server{
        Addr:    ":8080",
        Handler: router,
    }

    return &Server{
        httpServer: httpServer,
        scanner:    scanner,
        browser:    browser,
    }, nil
}

func (s *Server) Start() error {
    return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown() error {
    s.browser.Close()
    return s.httpServer.Shutdown(nil)
}