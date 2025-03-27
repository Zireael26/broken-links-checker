package scanner

import (
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

func (s *Scanner) ProcessStaticWebpage(task CrawlTask, tasks chan CrawlTask, results chan<- LinkResult, wg *sync.WaitGroup, visited *sync.Map, maxDepth int) {
	defer func () {
		log.Printf("ProcessStaticWebpage: Done processing task for %s", task.URL)
		wg.Done()
	}()
    // Fetch the page
	log.Printf("Fetching %s", task.URL)
	req, _ := http.NewRequest(http.MethodGet, task.URL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
    resp, err := s.client.Do(req)
    if err != nil {
        log.Printf("Failed to fetch %s: %v", task.URL, err)
        results <- LinkResult{URL: task.URL, Status: "Failed to fetch", Code: 0, Depth: task.Depth}
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode > 299 {
        log.Printf("Failed to fetch %s: %s", task.URL, resp.Status)
        results <- LinkResult{URL: task.URL, Status: resp.Status, Code: resp.StatusCode, Depth: task.Depth}
        return
    }

	// First add the current page to the results
	results <- LinkResult{URL: task.URL, Status: resp.Status, Code: resp.StatusCode, Depth: task.Depth}

    // Parse the page
    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        log.Printf("Failed to parse %s: %v", task.URL, err)
        return
    }

    // Extract various types of links concurrently
    var extractWg sync.WaitGroup
    links := make(map[string]string)
    var mu sync.Mutex
    for query, attr := range LinkTypes {
        extractWg.Add(1)
        go func(query, attr string) {
            defer extractWg.Done()
            extractLinks(query, attr, doc, links, &mu)
        }(query, attr)
    }
    extractWg.Wait()
	preppedLinks := prepareLinks(task, links, visited)

	if (task.Depth == maxDepth) {
		wg.Add(len(preppedLinks))
	}
	// Process the links in the following way. 
	// If the current depth is less than max depth, create a new task for each link and send it to the tasks channel.
	// If the current depth is equal to max depth (leaf node, check the status of each link and send it to the results channel.
	for link, text := range preppedLinks {
		if task.Depth < maxDepth {
			// TODO: Improve the check for links starting with the base URL
			if (isInternalURL(task.BaseURL, link)) {
				wg.Add(1)
				tasks <- CrawlTask{BaseURL: task.BaseURL, URL: link, Depth: task.Depth + 1}
			}
		} else if task.Depth == maxDepth {
			s.checkLeafNode(link, text, task.Depth, results, visited)
			wg.Done()
		}
	}
}

func extractLinks(query string, attr string, doc *goquery.Document, links map[string]string, mu *sync.Mutex) {
    doc.Find(query).Each(func(i int, sel *goquery.Selection) {
        link, _ := sel.Attr(attr)
        link = strings.TrimSpace(link)
        if link == "" {
            return
        }
        mu.Lock()
        links[link] = sel.Text()
        mu.Unlock()
    })
}
