package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/gocolly/colly"
	"github.com/logrusorgru/aurora"
)

const (
	DEFAULT_USER_AGENT                   = "Simple_Link_Health_BOT"
	DEFAULT_HEALTHY_HTTP_MIN_STATUS_CODE = 200
	DEFAULT_HEALTHY_HTTP_MAX_STATUS_CODE = 299
)

func main() {
	userAgent := flag.String("userAgent", DEFAULT_USER_AGENT, "User-Agent")
	depth := flag.Int("depth", 2, "Max depth")
	threads := flag.Int("threads", 4, "Number of threads to use")
	url := flag.String("url", "", "URL to use")

	flag.Parse()
	targetURL, urlError := getURL(*url)
	if urlError != nil {
		handleFatal(urlError)
	}
	collector := getCollector(*userAgent, *depth, *threads)
	collectorError := collector.Visit(targetURL.String())
	if collectorError != nil {
		handleError(collectorError)
	}
	collector.Wait()
}

// Represents a requested link containing the url and status derived from the requests response.
type Link struct {
	status int
	url    *url.URL
}

// Checks whether the link was healthy by using the link status
func (link *Link) isHealthy() bool {
	return link.status >= DEFAULT_HEALTHY_HTTP_MIN_STATUS_CODE && link.status <= DEFAULT_HEALTHY_HTTP_MAX_STATUS_CODE
}

// Prints the link status, and formats the output color based on link health
func (link *Link) printLinkStatus(isHealthy bool) {
	if isHealthy {
		fmt.Printf(
			"%s	%s\n",
			link.url,
			aurora.Green("healthy"),
		)
	} else {
		fmt.Printf(
			"%s	%s	%d\n",
			link.url,
			aurora.Red("down"),
			aurora.Bold(link.status),
		)
	}
}

// Attempts to parse the provided URL, returns an instance of URL if it is valid otherwise returns null
func getURL(targetURL string) (*url.URL, error) {
	if !isValidURL(targetURL) {
		return nil, fmt.Errorf("Invalid URL")
	}

	return url.Parse(targetURL)
}

// Helper function to validate the provided URL
func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// Initializes a new collector instance
func getCollector(userAgent string, depth int, threads int) *colly.Collector {
	collector := colly.NewCollector(
		colly.Async(true),
		colly.UserAgent(userAgent),
		colly.MaxDepth(depth),
		colly.URLFilters(
			regexp.MustCompile("https?://.+$"),
		),
	)

	limitError := collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: threads,
		RandomDelay: 1 * time.Second,
	})

	if limitError != nil {
		handleError(limitError)
	}

	// On error print the reason the request failed
	collector.OnError(func(response *colly.Response, err error) {
		url := response.Request.URL
		reason := err.Error()

		if reason == "" {
			reason = "Unknown"
		}

		handleError(fmt.Errorf("Request to %s failed. Reason: %s", url, reason))
	})

	collector.OnHTML("a[href]", func(element *colly.HTMLElement) {
		link := element.Attr("href")
		_ = element.Request.Visit(link)
	})

	collector.OnResponse(func(response *colly.Response) {
		link := Link{
			url:    response.Request.URL,
			status: response.StatusCode,
		}

		if !link.isHealthy() {
			link.printLinkStatus(false)
			return
		}

		link.printLinkStatus(true)
	})

	return collector
}

func handleError(error error) {
	if error != nil {
		fmt.Println(aurora.Red("Error:"), error)
	}
}

func handleFatal(error error) {
	if error != nil {
		fmt.Println(aurora.BrightRed("Fatal:"), error)
		os.Exit(1)
	}
}
