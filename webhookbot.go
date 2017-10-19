package main

import (
	"fmt"
	"time"
	"net/http"
	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"github.com/spf13/viper"
	"github.com/mmcdole/gofeed"
	"encoding/json"
	"bytes"
)

var (
	webhookUrl	string
)


func parseBugBountyForum(checkDate time.Time) {
	url := "https://bugbountyforum.com/blogs/"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	articles := scrape.FindAll(root, scrape.ByClass("article"))

	for _, article := range articles {
		// Pull the useful information from each article section
		title, _ := scrape.Find(article, scrape.ByClass("title"))
		titleText := scrape.Text(title)
		date, _ := scrape.Find(article, scrape.ByClass("date"))
		dateText := scrape.Text(date)
		summary, _ := scrape.Find(article, scrape.ByClass("summary"))
		summaryText := scrape.Text(summary)
		url, _:= scrape.Find(article, scrape.ByTag(atom.A))
		urlText := "https://bugbountyforum.com" + scrape.Attr(url, "href")

		// Parse the date and see if this article is new
		dateFormat := "Jan 2, 2006"
		parsedDate, _ := time.Parse(dateFormat, dateText)

		// Check to see if the article was published after the last time we checked
		if parsedDate.Before(checkDate) {
			// This is where we ship a link to Discord Webhooks
			postWebhook(titleText, "Bug Bounty Forum", summaryText, urlText, dateText)
		}
	}
}

func postWebhook(title string, source string, summary string, url string, date string) int {
	titleUrl := "**[" + title + "]("+ url + ")**"
	summaryString := "```" + date + "\n" + summary + "```"
	content := titleUrl + "\n" + summaryString
	jsonMap := map[string]string{"content": content, "username": source}
	jsonData, _ := json.Marshal(jsonMap)
	request, _ := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	response, err := client.Do(request)

	if err != nil {
		panic(err)
	}

	return response.StatusCode
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
	// Setup Configuration
	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	webhookUrl = viper.GetString("webhookUrl")

	// This is where we execute all of our checkers
	currentTime := time.Now()
	parseBugBountyForum(currentTime)
}
