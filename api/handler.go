package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Zireael26/broken-links-checker/internal/report"
	"github.com/Zireael26/broken-links-checker/internal/scanner"
)

// PerformanceMetrics represents scan performance statistics
// Copied from report package to avoid circular dependencies
type PerformanceMetrics struct {
    StaticScans            int64 `json:"static_scans"`
    SpoofedScans           int64 `json:"spoofed_scans"`
    RenderedScans          int64 `json:"rendered_scans"`
    EscalationsToSpoofed   int64 `json:"escalations_to_spoofed"`
    EscalationsToRendered  int64 `json:"escalations_to_rendered"`
    TimeoutErrors          int64 `json:"timeout_errors"`
    NetworkErrors          int64 `json:"network_errors"`
    RetryAttempts          int64 `json:"retry_attempts"`
    SuccessfulRetries      int64 `json:"successful_retries"`
}

type Handler struct {
	scanner *scanner.Scanner
	scans   map[string]chan scanner.LinkResult // In-memory scan tracking
}

func NewHandler(s *scanner.Scanner) *Handler {
	return &Handler{
		scanner: s,
		scans:   make(map[string]chan scanner.LinkResult),
	}
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL          string `json:"url"`
		Depth        int    `json:"depth,omitempty"`
		Workers      int    `json:"workers,omitempty"`
		RateLimit    int    `json:"rate_limit,omitempty"`
		DomainOnly   bool   `json:"domain_only,omitempty"`
		SiteType     string `json:"site_type,omitempty"`
		Retries      int    `json:"retries,omitempty"`
		RetryDelay   int    `json:"retry_delay,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Configure scanner with provided options
	config := scanner.ScanConfig{
		MaxWorkers: req.Workers,
		MaxDepth:   req.Depth,
		RateLimit:  req.RateLimit,
		DomainOnly: req.DomainOnly,
		SiteType:   req.SiteType,
		Retries:    req.Retries,
		RetryDelay: req.RetryDelay,
	}

	// Apply defaults if not provided
	if config.MaxDepth <= 0 {
		config.MaxDepth = 2 // Default depth
	}
	
	if config.MaxDepth > 3 {
		config.MaxDepth = 3 // Cap depth at 3 for safety
	}

	if config.MaxWorkers <= 0 {
		config.MaxWorkers = 50 // Default workers
	}
	
	if config.MaxWorkers > 100 {
		config.MaxWorkers = 100 // Cap workers at 100
	}
	
	if config.Retries <= 0 {
		config.Retries = 2 // Default retry count
	}
	
	if config.RetryDelay <= 0 {
		config.RetryDelay = 1000 // Default retry delay (ms)
	}
	
	// Configure scanner
	h.scanner.Configure(config)

	scanID := generateScanID()
	tasks := make(chan scanner.CrawlTask, 100)
	results := make(chan scanner.LinkResult, 100)
	h.scans[scanID] = results

	var swg sync.WaitGroup
	swg.Add(1)
	go func() {
		log.Printf("Starting scan %s for %s with depth %d", scanID, req.URL, config.MaxDepth)
		h.scanner.Scan(req.URL, config.MaxDepth, tasks, results, &swg)
		reportPath := filepath.Join("reports", scanID+".json")
		
		swg.Wait()
		close(tasks)
		close(results)
		log.Printf("Scan %s completed, len(results): %d", scanID, len(results))
		// Get metrics before saving the report
		metrics := h.scanner.GetMetrics()
		if err := report.SaveReport(scanID, req.URL, config.MaxDepth, results, reportPath, metrics); err != nil {
			log.Printf("Failed to save report for %s: %v", scanID, err)
		}
		log.Printf("Report saved to %s", reportPath)
		defer delete(h.scans, scanID)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"scan_id": scanID,
		"status":  "started",
	})
}

// GetScanResults retrieves scan results with customizable output format
func (h *Handler) GetScanResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the scan ID from URL path
	scanID := strings.TrimPrefix(r.URL.Path, "/scan/")
	if scanID == "" {
		http.Error(w, "Scan ID required", http.StatusBadRequest)
		return
	}

	// Check if scan is still running
	w.Header().Set("Content-Type", "application/json")
	if _, exists := h.scans[scanID]; exists {
		json.NewEncoder(w).Encode(map[string]string{
			"scan_id": scanID,
			"status":  "running",
		})
		return
	}

	// Build the path to the report file
	reportPath := filepath.Join("reports", scanID+".json")
	log.Printf("Loading report from %s", reportPath)

	// Check if report exists
	file, err := os.Open(reportPath)
	if err != nil {
		http.Error(w, "Scan report not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Parse the report JSON
	var report report.Report
	if err := json.NewDecoder(file).Decode(&report); err != nil {
		log.Printf("Failed to parse report for %s: %v", scanID, err)
		http.Error(w, "Failed to parse report", http.StatusInternalServerError)
		return
	}

	// Get the requested output format from query parameters
	// Default is "summary" if not specified
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "summary"
	}

	// Prepare the response based on the requested format
	switch format {
	case "summary":
		sendSummaryResponse(w, report)
	case "broken_links":
		sendBrokenLinksResponse(w, report)
	case "all_links":
		sendAllLinksResponse(w, report)
	default:
		http.Error(w, "Invalid format parameter. Must be one of: summary, broken_links, all_links", http.StatusBadRequest)
	}
}

// sendSummaryResponse sends a summary of the scan results
func sendSummaryResponse(w http.ResponseWriter, report report.Report) {
	summary := struct {
		ScanID           string             `json:"scan_id"`
		StartURL         string             `json:"start_url"`
		Timestamp        string             `json:"timestamp"`
		Duration         string             `json:"duration"`
		TotalLinks       int                `json:"total_links"`
		WorkingLinksCount int               `json:"working_links_count"`
		BrokenLinksCount int                `json:"broken_links_count"`
		Performance      PerformanceMetrics `json:"performance"`
	}{
		ScanID:           report.ScanID,
		StartURL:         report.StartURL,
		Timestamp:        report.Timestamp,
		Duration:         report.Duration,
		TotalLinks:       report.TotalLinks,
		WorkingLinksCount: len(report.WorkingLinks),
		BrokenLinksCount: len(report.BrokenLinks),
		Performance: PerformanceMetrics{
			StaticScans:           report.Performance.StaticScans,
			SpoofedScans:          report.Performance.SpoofedScans,
			RenderedScans:         report.Performance.RenderedScans,
			EscalationsToSpoofed:  report.Performance.EscalationsToSpoofed,
			EscalationsToRendered: report.Performance.EscalationsToRendered,
			TimeoutErrors:         report.Performance.TimeoutErrors,
			NetworkErrors:         report.Performance.NetworkErrors,
			RetryAttempts:         report.Performance.RetryAttempts,
			SuccessfulRetries:     report.Performance.SuccessfulRetries,
		},
	}

	json.NewEncoder(w).Encode(summary)
}

// sendBrokenLinksResponse sends only the broken links from the scan
func sendBrokenLinksResponse(w http.ResponseWriter, report report.Report) {
	result := struct {
		ScanID      string               `json:"scan_id"`
		StartURL    string               `json:"start_url"`
		Timestamp   string               `json:"timestamp"`
		BrokenLinks []scanner.LinkResult `json:"broken_links"`
		Count       int                  `json:"count"`
	}{
		ScanID:      report.ScanID,
		StartURL:    report.StartURL,
		Timestamp:   report.Timestamp,
		BrokenLinks: report.BrokenLinks,
		Count:       len(report.BrokenLinks),
	}

	json.NewEncoder(w).Encode(result)
}

// sendAllLinksResponse sends all links from the scan
func sendAllLinksResponse(w http.ResponseWriter, report report.Report) {
	// Return the complete report as is
	json.NewEncoder(w).Encode(report)
}

func generateScanID() string {
	return time.Now().Format("20060102T150405") // Simple timestamp-based ID
}
