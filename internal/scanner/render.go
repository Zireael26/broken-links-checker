package scanner

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// ProcessSpoofedWebpage loads a page with Googlebot user agent for prerendered content
func (s *Scanner) ProcessSpoofedWebpage(task CrawlTask, tasks chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
	defer func() {
		log.Printf("ProcessSpoofedWebpage: Done processing task for %s", task.URL)
		wg.Done()
	}()

	log.Printf("Spoofed fetching %s", task.URL)
	
	// Create a new page with Googlebot user agent
	page := s.browser.MustPage().MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	})
	defer page.Close()

	// Set timeout for navigation
	err := page.Timeout(10 * time.Second).Navigate(task.URL)
	if err != nil {
		log.Printf("Failed to navigate to %s: %v", task.URL, err)
		results <- LinkResult{
			URL:    task.URL,
			Status: "Failed to fetch",
			Code:   0,
			Depth:  task.Depth,
			Tier:   "spoofed",
		}
		return
	}

	// Wait for network to be idle (most prerendered content should be loaded)
	// Corrected: WaitNavigation returns a function that needs to be called
	waitFunc := page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)
	waitErr := rod.Try(func() {
		waitFunc()
	})
	if waitErr != nil {
		log.Printf("Network timeout for %s: %v", task.URL, waitErr)
	}

	// First add the current page to the results
	status := "OK"
	statusCode := 200
	if page.MustInfo().URL != task.URL && !strings.HasPrefix(page.MustInfo().URL, task.URL) {
		// Redirected
		status = "Redirected"
	}

	results <- LinkResult{
		URL:    task.URL,
		Status: status,
		Code:   statusCode,
		Depth:  task.Depth,
		Tier:   "spoofed",
	}

	// Extract links from the page
	links, err := extractLinksFromPage(page)
	if err != nil {
		log.Printf("Failed to extract links from %s: %v", task.URL, err)
		return
	}
	
	// Apply domain filtering if enabled
	links = s.filterLinksByDomain(task.BaseURL, links)

	preppedLinks := prepareLinks(task, links, visited)

	if task.Depth == maxDepth {
		wg.Add(len(preppedLinks))
	}

	// Process the links
	for link, text := range preppedLinks {
		if task.Depth < maxDepth {
			if isInternalURL(task.BaseURL, link) {
				wg.Add(1)
				tasks <- CrawlTask{BaseURL: task.BaseURL, URL: link, Depth: task.Depth + 1}
			}
		} else if task.Depth == maxDepth {
			s.checkLeafNode(link, text, task.Depth, results, visited)
			wg.Done()
		}
	}
}

// ProcessDynamicWebpage uses full browser rendering with JavaScript execution
func (s *Scanner) ProcessDynamicWebpage(task CrawlTask, tasks chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
	defer func() {
		log.Printf("ProcessDynamicWebpage: Done processing task for %s", task.URL)
		wg.Done()
	}()

	log.Printf("Dynamic rendering %s", task.URL)
	
	// Create a new page with modern browser user agent
	page := s.browser.MustPage()
	defer page.Close()

	// Set timeout for navigation
	err := page.Timeout(15 * time.Second).Navigate(task.URL)
	if err != nil {
		log.Printf("Failed to navigate to %s: %v", task.URL, err)
		results <- LinkResult{
			URL:    task.URL,
			Status: "Failed to fetch", 
			Code:   0,
			Depth:  task.Depth,
			Tier:   "rendered",
		}
		return
	}

	// Wait for network to be idle and additional time for JavaScript to execute
	// Corrected: WaitNavigation returns a function that needs to be called
	waitFunc := page.WaitNavigation(proto.PageLifecycleEventNameNetworkAlmostIdle)
	waitErr := rod.Try(func() {
		waitFunc()
	})
	if waitErr != nil {
		log.Printf("Network timeout for %s: %v", task.URL, waitErr)
	}
	
	// Wait for any JavaScript animations or async content to load
	// Corrected: MustWait expects a JavaScript expression as a string, not a duration
	page.MustEval(`() => new Promise(r => setTimeout(r, 500))`)

	// First add the current page to the results
	status := "OK"
	statusCode := 200
	if page.MustInfo().URL != task.URL && !strings.HasPrefix(page.MustInfo().URL, task.URL) {
		// Redirected
		status = "Redirected"
	}

	results <- LinkResult{
		URL:    task.URL,
		Status: status,
		Code:   statusCode,
		Depth:  task.Depth,
		Tier:   "rendered",
	}

	// Extract links from the page (including dynamically rendered content)
	links, err := extractLinksFromPage(page)
	if err != nil {
		log.Printf("Failed to extract links from %s: %v", task.URL, err)
		return
	}
	
	// Apply domain filtering if enabled
	links = s.filterLinksByDomain(task.BaseURL, links)

	preppedLinks := prepareLinks(task, links, visited)

	if task.Depth == maxDepth {
		wg.Add(len(preppedLinks))
	}

	// Process the links
	for link, text := range preppedLinks {
		if task.Depth < maxDepth {
			if isInternalURL(task.BaseURL, link) {
				wg.Add(1)
				tasks <- CrawlTask{BaseURL: task.BaseURL, URL: link, Depth: task.Depth + 1}
			}
		} else if task.Depth == maxDepth {
			s.checkLeafNode(link, text, task.Depth, results, visited)
			wg.Done()
		}
	}
}

// extractLinksFromPage extracts links from a rod page
func extractLinksFromPage(page *rod.Page) (map[string]string, error) {
	links := make(map[string]string)

	// Extract links using JavaScript
	// This covers all the types of links we're looking for in one go
	linksObj, err := page.Eval(`() => {
		const links = {};
		
		// Handle <a> elements
		document.querySelectorAll('a[href]').forEach(el => {
			links[el.href] = el.textContent.trim();
		});
		
		// Handle <link> elements
		document.querySelectorAll('link[href]').forEach(el => {
			links[el.href] = el.rel || '';
		});
		
		// Handle elements with src attributes
		['script', 'img', 'source', 'video', 'audio', 'iframe', 'embed', 'track'].forEach(tag => {
			document.querySelectorAll(tag + '[src]').forEach(el => {
				links[el.src] = el.alt || el.title || '';
			});
		});
		
		// Handle object[data]
		document.querySelectorAll('object[data]').forEach(el => {
			links[el.data] = el.type || '';
		});
		
		// Handle param[value]
		document.querySelectorAll('param[value]').forEach(el => {
			links[el.value] = el.name || '';
		});
		
		return links;
	}`)

	if err != nil {
		return nil, err
	}

	// Convert the JavaScript object to a Go map
	for k, v := range linksObj.Value.Map() {
		links[k] = v.Str()
	}

	return links, nil
}