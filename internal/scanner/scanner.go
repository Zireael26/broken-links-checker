package scanner

import (
	"net/http"
	"sync"

	"github.com/go-rod/rod"
)

type Scanner struct {
    client  *http.Client
    browser *rod.Browser
}

func New(client *http.Client, browser *rod.Browser) *Scanner {
    return &Scanner{client: client, browser: browser}
}

func (s *Scanner) Scan(url string, maxDepth, maxWorkers int, results chan<- LinkResult) {
    tasks := make(chan CrawlTask, 100)
    var wg sync.WaitGroup
    visited := &sync.Map{}

    for i := 0; i < maxWorkers; i++ {
        wg.Add(1)
        go s.worker(tasks, results, &wg, visited, maxDepth)
    }

    tasks <- CrawlTask{URL: url, Depth: 0}
    go func() {
        wg.Wait()
        close(tasks)
        close(results)
    }()
}

func (s *Scanner) worker(tasks <-chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
    defer wg.Done()
    // Stub: Add static.go, spoofed.go, rendered.go logic here
    for task := range tasks {
        if _, loaded := visited.LoadOrStore(task.URL, true); loaded || task.Depth > maxDepth {
            continue
        }

		// Stub: Add the multi-tier logic here
		// Stub: Add leaf node checking logic here
        results <- LinkResult{URL: task.URL, Status: "OK", Code: 200, Depth: task.Depth, Tier: "stub"}
    }
}