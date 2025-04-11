package report

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Zireael26/broken-links-checker/internal/scanner"
)

type Report struct {
    ScanID       string             `json:"scan_id"`
    StartURL     string             `json:"start_url"`
    Timestamp    string             `json:"timestamp"`
    Depth        int                `json:"depth"`
    TotalLinks   int                `json:"total_links"`
    WorkingLinks []scanner.LinkResult `json:"working_links"`
    BrokenLinks  []scanner.LinkResult `json:"broken_links"`
    Performance  PerformanceMetrics `json:"performance,omitempty"`
    Duration     string             `json:"duration,omitempty"` // Total scan duration
}

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

func SaveReport(scanID, startURL string, depth int, results <-chan scanner.LinkResult, path string, metrics scanner.ScanMetrics) error {
    var working, broken []scanner.LinkResult
    
    // Track the start time to calculate duration
    startTime := time.Now()
    
    for result := range results {
		// Accept all 2xx codes as working
        if result.Code < 299 && result.Code >= 200 {
            working = append(working, result)
        } else {
            broken = append(broken, result)
        }
    }

    // Calculate duration
    duration := time.Since(startTime)

    // Create performance metrics
    performanceMetrics := PerformanceMetrics{
        StaticScans:           metrics.StaticScans,
        SpoofedScans:          metrics.SpoofedScans,
        RenderedScans:         metrics.RenderedScans,
        EscalationsToSpoofed:  metrics.EscalationsToSpoofed,
        EscalationsToRendered: metrics.EscalationsToRendered,
        TimeoutErrors:         metrics.TimeoutErrors,
        NetworkErrors:         metrics.NetworkErrors,
        RetryAttempts:         metrics.RetryAttempts,
        SuccessfulRetries:     metrics.SuccessfulRetries,
    }

    report := Report{
        ScanID:       scanID,
        StartURL:     startURL,
        Timestamp:    time.Now().Format(time.RFC3339),
        Depth:        depth,
        TotalLinks:   len(working) + len(broken),
        WorkingLinks: working,
        BrokenLinks:  broken,
        Performance:  performanceMetrics,
        Duration:     duration.String(),
    }

    // Create reports directory if it doesn't exist
    os.MkdirAll("reports", os.ModePerm)

    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()

    enc := json.NewEncoder(file)
    enc.SetIndent("", "  ")
    return enc.Encode(report)
}