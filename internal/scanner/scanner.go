package scanner

import (
	"log"
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

func (s *Scanner) Scan(url string, maxDepth int, results chan<- LinkResult, swg *sync.WaitGroup) {
    tasks := make(chan CrawlTask, 100)
    // var wg sync.WaitGroup
    visited := &sync.Map{}
    tasks <- CrawlTask{URL: url, Depth: 0}

	log.Printf("Added initial task for %s", url)
	go s.worker(tasks, results, swg, visited, maxDepth)

    // defer close(tasks)
}

func (s *Scanner) worker(tasks chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
    // TODO: Add spoofed.go, rendered.go logic
	log.Printf("Found %d tasks", len(tasks))
    for task := range tasks {
		if _, loaded := visited.LoadOrStore(task.URL, true); loaded || task.Depth > maxDepth {
			log.Printf("Skipping task for %s", task.URL)
            continue
        }
		
		// TODO: Add the multi-tier logic here
		log.Printf("Processing task for %s", task.URL)
		s.ProcessStaticWebpage(task, tasks, results, wg, visited, maxDepth)
    }
}

func (s *Scanner) checkLeafNode(url, ref string, depth int, results chan<- LinkResult, visited *sync.Map) {
	log.Printf("Checking leaf node: %s", url)
	resp, err := s.client.Head(url)
	_, loaded := visited.LoadOrStore(url, true)
	if loaded{
		return
	}

	if err != nil {
		results <- LinkResult{URL: url, Ref: ref, Status: "Failed to fetch", Code: 0, Depth: depth}
		return
	}

	if resp.StatusCode > 299 {
		results <- LinkResult{URL: url, Ref: ref, Status: resp.Status, Code: resp.StatusCode, Depth: depth}
		return
	}

	results <- LinkResult{URL: url, Ref: ref, Status: resp.Status, Code: resp.StatusCode, Depth: depth}
}