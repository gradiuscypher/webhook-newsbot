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
)

var (
	webhookUrl	string
)

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
	statement.Exec(currentTime.String(), source)
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
	timeLayout := "2006-01-02 15:04:05 -0700 MST"
	fmt.Println("Now parsing:", lastDateStr)
	lastDate, _ = time.Parse(timeLayout, lastDateStr)
	return lastDate
}

func parseBugBountyForum() {
	sourceStr := "bugbountyforum"

	// Get the last BugBountyFourm post date that was processed
	lastUpdate := getLastPostDate(sourceStr)

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
		if parsedDate.After(lastUpdate) {
			// This is where we ship a link to Discord Webhooks
			postWebhook(titleText, "Bug Bounty Forum", summaryText, urlText, dateText)
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
	//parseBugBountyForum()
	//parseHackerOneDisclosure()
	//rssParser()
}
