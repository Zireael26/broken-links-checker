package scanner

import (
	"fmt"
	"log"
	"strings"
	"sync"
)


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

func isInternalURL(baseURL, url string) bool {
	return strings.HasPrefix(url, baseURL)
}

func prepareURL(baseURL, url string) (cleanedURL string, err error) {
	if !strings.HasPrefix(url, "http") && !strings.HasPrefix(url, "https") && !strings.HasPrefix(url, "/") {
		return "", fmt.Errorf("URL %s is not valid", url)
	}

	cleanedURL = strings.Split(url, "#")[0]
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