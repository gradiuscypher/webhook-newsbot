package main

import (
	"fmt"
	"net/http"
	"golang.org/x/net/html"
	"github.com/mmcdole/gofeed"
)

func parseHtml(url string) {
	// reference: https://schier.co/blog/2015/04/26/a-simple-web-scraper-in-go.html
	// currently used to parse bugbountyforum blog
	resp, _ := http.Get(url)
	tokens := html.NewTokenizer(resp.Body)

	for {
		token := tokens.Next()

		switch {
		case token == html.ErrorToken:
			return
		case token == html.StartTagToken:
			tData := tokens.Token()

			isDiv := tData.Data == "div"

			if isDiv {
				for _, div := range tData.Attr {
					if div.Key == "class" {
						if div.Val == "date"{
						}
					}
				}
			}
		}
	}
}

func rssParser()  {
	// Example RSS Parsing
	feedParser := gofeed.NewParser()
	feed, _ := feedParser.ParseURL("https://threatpost.com/feed/")
	fmt.Println(feed.Title)
	fmt.Println(feed.UpdatedParsed)

	for _, item := range feed.Items {
		fmt.Println(item.Title + " [" + item.PublishedParsed.String() + "]")
		fmt.Println("===========")
		fmt.Println(item.Description + "\n")
	}
}

func main() {
	parseHtml("https://bugbountyforum.com/blogs/")
}
