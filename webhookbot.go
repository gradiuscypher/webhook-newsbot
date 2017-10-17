package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
)

func main() {
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
