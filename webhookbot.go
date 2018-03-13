package main

import (
	"fmt"
	"time"
	"net/http"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"github.com/spf13/viper"
	"github.com/mmcdole/gofeed"
	"github.com/yhat/scrape"
	"encoding/json"
	"bytes"
	"database/sql"
	"log"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
)

var (
	webhookUrl	string
)

type Footer struct {
	Text string`json:"text"`
	IconUrl string`json:"icon_url"`
	ProxyIconUrl string`json:"proxy_icon_url"`
}

type EmbedField struct {
	Name string`json:"name"`
	Value string`json:"value"`
	Inline bool`json:"inline"`
}

type EmbedThumbnail struct {
	Url string`json:"url"`
}

type MessageEmbed struct {
	Title string`json:"title"`
	Description string`json:"description"`
	Url string`json:"url"`
	Color int64`json:"color"`
	//Fields []EmbedField`json:"fields"`
	EmbedThumbnail EmbedThumbnail`json:"thumbnail"`
	Footer Footer`json:"footer"`
}

type WebhookEmbedMessage struct {
	Embed []MessageEmbed`json:"embeds"`
	Username string`json:"username"`
}

type RssFeed struct {
	FeedUrl string`json:"feed_url"`
	FeedName string`json:"feed_name"`
	FeedIconUrl string`json:"feed_icon_url"`
}

func updateLastPostDate(source string) {
	currentTime := time.Now()

	// Setup the SQLite DB for tracking posts
	db, err := sql.Open("sqlite3", "posts.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	statement, err := db.Prepare("UPDATE posts SET lastDate = ? WHERE source = ?")
	if err != nil {
		log.Fatal(err)
	}
	statement.Exec(currentTime.Format(time.UnixDate), source)
}

func getLastPostDate(source string) (time.Time) {
	var lastDate time.Time
	var lastDateStr string
	currentTime := time.Now()

	// Setup the SQLite DB for tracking posts
	db, err := sql.Open("sqlite3", "posts.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create the DB if it doesn't exist
	db.Exec("CREATE TABLE IF NOT EXISTS posts (id integer PRIMARY KEY, source text NOT NULL, lastDate text)")
	if err != nil {
		log.Fatal(err)
	}

	// Look up the last post date
	err = db.QueryRow("SELECT lastDate FROM posts WHERE source = ?", source).Scan(&lastDateStr)
	if err == sql.ErrNoRows {
		// If no date was found, return current date and save to DB
		statement, err := db.Prepare("INSERT INTO posts(source, lastDate) VALUES(?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		statement.Exec(source, currentTime.String())
		return currentTime

	} else if err != nil {
		// Something broke and it wasn't a nil date string
		log.Fatal(err)
	}

	// Return the last found date string
	//timeLayout := "2006-01-02 15:04:05 -0700 MST"
	//lastDate, _ = time.Parse(timeLayout, lastDateStr)
	lastDate, _ = time.Parse(time.UnixDate, lastDateStr)
	return lastDate
}

func parseBugBountyForum() {
	sourceStr := "bugbountyforum"

	// Get the last BugBountyFourm post date that was processed
	lastUpdate := getLastPostDate(sourceStr)

	url := "https://bugbountyforum.com/blogs/"
	imageUrl := "https://pbs.twimg.com/profile_images/941769675611836416/xiEzOny3_400x400.jpg"
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
		if parsedDate.After(lastUpdate) {
			// This is where we ship a link to Discord Webhooks
			postWebhookEmbed(titleText, "Bug Bounty Fourm", summaryText, urlText, dateText, imageUrl)
		}
	}
	updateLastPostDate(sourceStr)
}

func parseHackerOneDisclosure() {
	//source := "hackerone"
	// TODO: Need to use something like PhantomGo to load the Javascript

	url := "https://hackerone.com/hacktivity?sort_type=latest_disclosable_activity_at&filter=type%3Apublic&page=1&range=forever"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println(root)

	disclosures := scrape.FindAll(root, scrape.ByClass("hacktivity__wrapper"))

	for _, disclosure := range disclosures {
		fmt.Println(disclosure)
	}
}

func parseRssFeeds() {
	// Parse the feed settings from the json file
	var feedList []RssFeed
	jsonFile, err := ioutil.ReadFile("feeds.json")
	if err != nil {
		panic(err)
	}
	json.Unmarshal(jsonFile, &feedList)

	// Iterate through each item in the feed list
	for _, item := range feedList {
		rssParser(item.FeedUrl, item.FeedIconUrl, item.FeedName)
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

func postWebhookEmbed(title string, source string, summary string, url string, date string, imageUrl string) int {
	if len(summary) > 256 {
		summary = summary[0:256] + "[...]"
	}

	footer := Footer {
		Text: date,
	}
	embedImage := EmbedThumbnail{
		Url: imageUrl,
	}

	embedList := []MessageEmbed {
		{
			Title: title,
			Url: url,
			Description: summary,
			Color: 4504154,
			Footer: footer,
			EmbedThumbnail: embedImage,
		},
	}

	whMessage := WebhookEmbedMessage {
		Embed: embedList,
		Username: source,
	}

	jsonData, _ := json.Marshal(whMessage)
	request, _ := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	response, err := client.Do(request)

	if err != nil {
		panic(err)
	}

	// Try it one more time if it didn't work, but sleep beforehand
	if response.StatusCode != 204 {
		time.Sleep(2)

		client := http.Client{}
		client.Do(request)

		if err != nil {
			panic(err)
		}
	}

	return response.StatusCode
}

func rssParser(feedUrl string, feedIconUrl string, feedName string)  {
	// Example RSS Parsing
	feedParser := gofeed.NewParser()
	feed, _ := feedParser.ParseURL(feedUrl)
	lastUpdate := getLastPostDate(feed.Title)
	fmt.Println(feed.Title, lastUpdate)

	for _, item := range feed.Items {
		if item.PublishedParsed.After(lastUpdate) {
			postWebhookEmbed(feedName + " - " + html.UnescapeString(item.Title), feedName, item.Description, item.Link, item.PublishedParsed.String(), feedIconUrl)
			updateLastPostDate(feed.Title)
		}
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
	parseBugBountyForum()
	parseRssFeeds()
}
