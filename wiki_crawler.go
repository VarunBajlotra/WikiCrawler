package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/gocolly/colly"
	"time"
	"strings"
	"regexp"
	"strconv"
	"github.com/jmhodges/levigo"
	"encoding/json"
)

func main() {
	flag.Parse()

	args := flag.Args()
	fmt.Println("Root url for crawling is", args)
	if len(args) < 1 {
		fmt.Println("Please enter a root url")
		fmt.Println("Syntax: go run wiki_crawler.go <link_url>")
		os.Exit(1)
	}

	tStart := time.Now();

	// For BFS traversal of pages, a queue is required
	// So, using channel of strings whose behaviour would resemble a queue

	// Using two queues
	// queue: contains all the links
	// filteredQueue: contains the links that are not yet visited (visited links are removed)
	queue := make(chan string)
	filteredQueue := make(chan string)

	go func() { queue <- args[0] }()
	go filterQueue(queue, filteredQueue)

	// Introduced a bool channel to synchronize execution of concurrently running 
	done := make(chan bool)

	count := 0

	for i := 0; i < 1; i++ {
		// Running an async Goroutine
		go func() {
			for uri := range filteredQueue {
				enqueue(uri, queue)
				count++
				fmt.Println("Varun Amigo Wiki Crawler:      ", count, "web pages crawled in", time.Since(tStart))
			}
			// Signalling the end of execution for current url
			done <- true
		}()
	}
	// Signalling the end of main thread
	<-done
	os.Exit(2)
}

// Traverses through the unfiltered queue and adds only the urls that are unvisited to the filtered queue
func filterQueue(in chan string, out chan string) {
	var visited = make(map[string]bool)
	for val := range in {
		if !visited[val] {
			visited[val] = true
			out <- val
		}
	}
}

// For one url:
// 1. Find all the links in the web page and add them in the queue
// 2. Store the web page text corresponding to the url 
func enqueue(uri string, queue chan string) {
	fmt.Println("Crawling", uri)

	// A colly which only crawls through Wikipedia Pages
	collyCollector := colly.NewCollector(
        colly.AllowedDomains("en.wikipedia.org"),
    )

    // Colly Collector triggers when it comes across body of the HTML page
    collyCollector.OnHTML("body", func(e *colly.HTMLElement) {

		// Finding all the links in the current page and extracting their absolute URLs
		// Then adding the url to the queue to be visited later on
        links := e.ChildAttrs("a", "href")
		for _, link := range links {
			absoluteUrl := e.Request.AbsoluteURL(link)
			if uri != "" {
				go func() {
					queue <- absoluteUrl 
				}()
			}
		}

		text := e.Text
		storeData(uri, text)
		
    })

    collyCollector.Visit(uri)

}

func storeData(uri, text string) {
	// Open the db
	opts := levigo.NewOptions()
	opts.SetCache(levigo.NewLRUCache(3<<30))
	opts.SetCreateIfMissing(true)
	db, _ := levigo.Open("dictionary2", opts)


	// Reading from the db
	ro := levigo.NewReadOptions()
	// Writing to the db (Key -> keyword, Value -> url)
	wo := levigo.NewWriteOptions()

	counts := wordCount(text)
	for word, freq := range counts{
		fmt.Println(word, "=>", freq)
		data, _ := db.Get(ro, []byte(word))
		
		// Converting byte stream to array of strings
		strArray := []string{}
		json.Unmarshal(data, &strArray)

		// Appending current string to the array
		updatedUri := uri + "," + strconv.Itoa(freq)
		strArray = append(strArray, updatedUri)
		fmt.Println("Appending URL to", word)

		// Pushing the updated array in the database
		strStream, _ := json.Marshal(strArray)
		_ = db.Put(wo, []byte(word), strStream)
	}

	ro.Close()
	wo.Close()
	db.Close()

}

// Calculates frequency of every word in string
func wordCount(str string) map[string]int {
    wordList := strings.Fields(str)
    counts := make(map[string]int)
    for _, word := range wordList {

		// Converting word to lower case and removing special characters using regex expression
		word = strings.ToLower(word)
		reg, err := regexp.Compile("[^a-zA-Z0-9]+")
		if err == nil {
			word = reg.ReplaceAllString(word, "")
		}

        _, ok := counts[word]
        if ok {
            counts[word] += 1
        } else {
            counts[word] = 1
        }
    }
    return counts
}

// go run wiki_crawler.go https://en.wikipedia.org/wiki/Web_scraping