package scanner

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod"
)

// Scanner handles the scanning of websites for broken links
type Scanner struct {
    client      *http.Client
    browser     *rod.Browser
    metrics     *ScanMetrics
    rateLimiter chan struct{}
    config      ScanConfig
}

// ScanConfig holds configuration for the scanner
type ScanConfig struct {
    MaxWorkers  int
    MaxDepth    int
    RateLimit   int          // Requests per second (0 for unlimited)
    DomainOnly  bool         // Limit scan to starting domain only
    SiteType    string       // Force a specific scanning tier (static, spoofed, rendered)
    Retries     int          // Number of retries for failed requests
    RetryDelay  int          // Delay between retries in milliseconds
}

// ScanMetrics tracks performance metrics during scanning
type ScanMetrics struct {
    TotalLinks         int64
    StaticScans        int64
    SpoofedScans       int64
    RenderedScans      int64
    EscalationsToSpoofed int64
    EscalationsToRendered int64
    TimeoutErrors      int64
    NetworkErrors      int64
    RetryAttempts      int64
    SuccessfulRetries  int64
}

// New creates a new Scanner with the specified HTTP client and browser
func New(client *http.Client, browser *rod.Browser) *Scanner {
    return &Scanner{
        client: client, 
        browser: browser,
        metrics: &ScanMetrics{},
        config: ScanConfig{
            MaxWorkers: 50,
            MaxDepth: 2,
            RateLimit: 0,
            Retries: 2,
            RetryDelay: 1000,
        },
    }
}

// Configure sets scanner configuration parameters
func (s *Scanner) Configure(config ScanConfig) {
    s.config = config
    
    // Initialize rate limiter if specified
    if config.RateLimit > 0 {
        s.rateLimiter = make(chan struct{}, config.RateLimit)
        
        // Fill the rate limiter buffer
        for i := 0; i < config.RateLimit; i++ {
            s.rateLimiter <- struct{}{}
        }
    }
}

// Scan initiates the scanning process for the given URL
func (s *Scanner) Scan(url string, maxDepth int, tasks chan CrawlTask, results chan<- LinkResult, swg *sync.WaitGroup) {
    visited := &sync.Map{}
    tasks <- CrawlTask{BaseURL:url, URL: url, Depth: 0}

	log.Printf("Added initial task for %s", url)
	
	// Create a worker pool
	workerCount := s.config.MaxWorkers
	if workerCount <= 0 {
	    workerCount = 10
	}
	
	for i := 0; i < workerCount; i++ {
		go s.worker(tasks, results, swg, visited, maxDepth)
	}
}

// worker processes tasks from the queue, using the appropriate scanning method
func (s *Scanner) worker(tasks chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
    for task := range tasks {
		if _, loaded := visited.LoadOrStore(task.URL, true); loaded || task.Depth > maxDepth {
			log.Printf("Skipping task for %s", task.URL)
			wg.Done()
            continue
        }
		
		// Apply rate limiting if configured
        if s.rateLimiter != nil {
            token := <-s.rateLimiter
            defer func() { s.rateLimiter <- token }()
        }
        
		log.Printf("Processing task for %s at depth %d", task.URL, task.Depth)
		
		// Check if site type is forced in configuration
		if s.config.SiteType != "" {
		    switch s.config.SiteType {
		    case "static":
		        atomic.AddInt64(&s.metrics.StaticScans, 1)
		        s.ProcessStaticWebpage(task, tasks, results, wg, visited, maxDepth)
		    case "spoofed":
		        atomic.AddInt64(&s.metrics.SpoofedScans, 1)
		        s.ProcessSpoofedWebpage(task, tasks, results, wg, visited, maxDepth)
		    case "rendered":
		        atomic.AddInt64(&s.metrics.RenderedScans, 1)
		        s.ProcessDynamicWebpage(task, tasks, results, wg, visited, maxDepth)
		    default:
		        // Default to auto detection
		        s.processWithEscalation(task, tasks, results, wg, visited, maxDepth)
		    }
		    continue
		}
		
		// If no forced type, use hybrid approach with escalation
		s.processWithEscalation(task, tasks, results, wg, visited, maxDepth)
    }
}

// processWithEscalation implements the hybrid scanning approach with escalation logic
func (s *Scanner) processWithEscalation(task CrawlTask, tasks chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
    // First try with static parsing (fastest)
    log.Printf("Trying static parsing for %s", task.URL)
    
    // Create a separate results channel for counting links found by static parsing
    staticResults := make(chan LinkResult, 100)
    var staticWg sync.WaitGroup
    staticWg.Add(1)
    
    // Track the number of links found
    linkCount := 0
    go func() {
        for result := range staticResults {
            linkCount++
            results <- result
        }
    }()
    
    // Clone task for processing
    staticTask := CrawlTask{
        BaseURL: task.BaseURL,
        URL: task.URL,
        Depth: task.Depth,
        Type: "static",
    }
    
    // Process with static parser
    atomic.AddInt64(&s.metrics.StaticScans, 1)
    s.ProcessStaticWebpage(staticTask, tasks, staticResults, &staticWg, visited, maxDepth)
    staticWg.Wait()
    close(staticResults)
    
    // Check if we should escalate to spoofed crawler
    if needsEscalationToSpoofed(task.URL, linkCount) {
        log.Printf("Escalating to spoofed crawler for %s (found %d links with static parsing)", task.URL, linkCount)
        atomic.AddInt64(&s.metrics.EscalationsToSpoofed, 1)
        
        // Create a separate results channel for counting links found by spoofed parsing
        spoofedResults := make(chan LinkResult, 100)
        var spoofedWg sync.WaitGroup
        spoofedWg.Add(1)
        
        // Reset link count
        linkCount = 0
        go func() {
            for result := range spoofedResults {
                linkCount++
                results <- result
            }
        }()
        
        // Clone task for processing
        spoofedTask := CrawlTask{
            BaseURL: task.BaseURL,
            URL: task.URL,
            Depth: task.Depth,
            Type: "spoofed",
        }
        
        // Process with spoofed crawler
        atomic.AddInt64(&s.metrics.SpoofedScans, 1)
        s.ProcessSpoofedWebpage(spoofedTask, tasks, spoofedResults, &spoofedWg, visited, maxDepth)
        spoofedWg.Wait()
        close(spoofedResults)
        
        // Check if we should escalate to full rendering
        if needsEscalationToRendered(task.URL, linkCount) {
            log.Printf("Escalating to full rendering for %s (found %d links with spoofed crawler)", task.URL, linkCount)
            atomic.AddInt64(&s.metrics.EscalationsToRendered, 1)
            
            // Clone task for processing
            renderedTask := CrawlTask{
                BaseURL: task.BaseURL,
                URL: task.URL,
                Depth: task.Depth,
                Type: "rendered", 
            }
            
            // Process with full rendering
            atomic.AddInt64(&s.metrics.RenderedScans, 1)
            s.ProcessDynamicWebpage(renderedTask, tasks, results, wg, visited, maxDepth)
        }
    }
}

// needsEscalationToSpoofed determines if a URL should be escalated from static to spoofed
func needsEscalationToSpoofed(url string, linkCount int) bool {
    // Escalate if very few links were found
    if linkCount < 5 {
        return true
    }
    
    // Escalate for sites likely to be SPAs or JS-driven
    spaPatterns := []string{
        "react", "angular", "vue", "ember", "svelte", 
        "app", "dashboard", "admin", "portal", 
        ".io", ".app", "api",
    }
    
    for _, pattern := range spaPatterns {
        if strings.Contains(strings.ToLower(url), pattern) {
            return true
        }
    }
    
    return false
}

// needsEscalationToRendered determines if a URL should be escalated from spoofed to rendered
func needsEscalationToRendered(url string, linkCount int) bool {
    // Escalate if very few links were found with spoofing
    if linkCount < 5 {
        return true
    }
    
    // Escalate for sites that are definitely SPAs
    definitiveSpaPatterns := []string{
        "react", "angular", "vue", "ember", "svelte", 
        "dashboard", "admin", "app/", 
        "single-page",
    }
    
    for _, pattern := range definitiveSpaPatterns {
        if strings.Contains(strings.ToLower(url), pattern) {
            return true
        }
    }
    
    return false
}

// GetMetrics returns the current scanner metrics
func (s *Scanner) GetMetrics() ScanMetrics {
    return ScanMetrics{
        TotalLinks:             atomic.LoadInt64(&s.metrics.TotalLinks),
        StaticScans:            atomic.LoadInt64(&s.metrics.StaticScans),
        SpoofedScans:           atomic.LoadInt64(&s.metrics.SpoofedScans),
        RenderedScans:          atomic.LoadInt64(&s.metrics.RenderedScans),
        EscalationsToSpoofed:   atomic.LoadInt64(&s.metrics.EscalationsToSpoofed),
        EscalationsToRendered:  atomic.LoadInt64(&s.metrics.EscalationsToRendered),
        TimeoutErrors:          atomic.LoadInt64(&s.metrics.TimeoutErrors),
        NetworkErrors:          atomic.LoadInt64(&s.metrics.NetworkErrors),
        RetryAttempts:          atomic.LoadInt64(&s.metrics.RetryAttempts),
        SuccessfulRetries:      atomic.LoadInt64(&s.metrics.SuccessfulRetries),
    }
}
