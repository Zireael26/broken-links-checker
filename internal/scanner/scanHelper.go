package scanner

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)


func (s *Scanner) checkLeafNode(url, ref string, depth int, results chan<- LinkResult, visited *sync.Map) {
	log.Printf("Checking leaf node: %s", url)
	_, loaded := visited.LoadOrStore(url, true)
	if loaded{
		return
	}

	// Implement retry logic
	var resp *http.Response
	var err error
	
	for attempt := 0; attempt <= s.config.Retries; attempt++ {
		if attempt > 0 {
			atomic.AddInt64(&s.metrics.RetryAttempts, 1)
			time.Sleep(time.Duration(s.config.RetryDelay) * time.Millisecond)
			log.Printf("Retry attempt %d for %s", attempt, url)
		}
		
		resp, err = s.client.Head(url)
		
		if err == nil {
			if attempt > 0 {
				atomic.AddInt64(&s.metrics.SuccessfulRetries, 1)
			}
			break
		}
		
		// Track error types for metrics
		if strings.Contains(err.Error(), "timeout") {
			atomic.AddInt64(&s.metrics.TimeoutErrors, 1)
		} else {
			atomic.AddInt64(&s.metrics.NetworkErrors, 1)
		}
	}

	if err != nil {
		results <- LinkResult{URL: url, Ref: ref, Status: "Failed to fetch", Code: 0, Depth: depth, Tier: "static"}
		return
	}

	results <- LinkResult{URL: url, Ref: ref, Status: resp.Status, Code: resp.StatusCode, Depth: depth, Tier: "static"}
}

func isInternalURL(baseURL, linkURL string) bool {
	return strings.HasPrefix(linkURL, baseURL)
}

// isDomainMatch checks if the link URL belongs to the same domain as the base URL
func isDomainMatch(baseURL, linkURL string) bool {
	baseU, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	
	linkU, err := url.Parse(linkURL)
	if err != nil {
		return false
	}
	
	return baseU.Hostname() == linkU.Hostname()
}

func prepareURL(baseURL, urlStr string) (cleanedURL string, err error) {
	if !strings.HasPrefix(urlStr, "http") && !strings.HasPrefix(urlStr, "https") && !strings.HasPrefix(urlStr, "/") {
		return "", fmt.Errorf("URL %s is not valid", urlStr)
	}

	cleanedURL = strings.Split(urlStr, "#")[0]
	if strings.HasPrefix(cleanedURL, "/") {
		cleanedURL = baseURL + cleanedURL
	}

	return cleanedURL, nil
}

func prepareLinks(task CrawlTask, links map[string]string, visited *sync.Map) map[string]string {
	preppedLinks := make(map[string]string)
	for link, ref := range links {
		cleanLink, err := prepareURL(task.BaseURL, link)
		if err != nil {
			log.Printf("Failed to prepare URL %s: %v", link, err)
			continue
		}

		if _, ok := visited.Load(cleanLink); ok {
			continue
		}

		preppedLinks[cleanLink] = ref
	}
	
	return preppedLinks
}

// filterLinksByDomain applies the domain_only filter if configured
func (s *Scanner) filterLinksByDomain(baseURL string, links map[string]string) map[string]string {
	if !s.config.DomainOnly {
		return links // No filtering needed
	}
	
	filteredLinks := make(map[string]string)
	for url, ref := range links {
		if isDomainMatch(baseURL, url) {
			filteredLinks[url] = ref
		}
	}
	
	return filteredLinks
}