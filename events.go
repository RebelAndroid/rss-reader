package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"time"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/mmcdole/gofeed/rss"
)

// Updates every feed currently in the databse
func update_feeds(db *sql.DB) {
	rows, err := db.Query("SELECT url FROM feeds")
	if err != nil {
		panic(err.Error())
	}
	for rows.Next() {
		var url string
		rows.Scan(&url)
		update_feed(db, url)
	}
	if rows.Err() != nil {
		panic(rows.Err())
	}
}

// Updates a single feed
func update_feed(db *sql.DB, url string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		println(err.Error())
		panic("unable to construct request")
	}

	// If we've scanned this feed already, we want to send the If-Modified-Since header to
	// allow the feed server to save resources by not sending duplicate entries
	row := db.QueryRow("SELECT last_updated FROM feeds WHERE url=?", url)
	buf := ""
	err = row.Scan(&buf)
	if err == nil {
		last_updated, err := time.Parse(time.RFC3339, buf)
		if err != nil {
			panic(err.Error())
		}
		req.Header.Add("If-Modified-Since", last_updated.Format(time.RFC1123Z))
	}

	resp, err := client.Do(req)
	if err != nil {
		println(err.Error())
		panic("unable to get feed")
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return
	}

	fp := rss.Parser{}
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		println(err.Error())
		panic("feed is malformed")
	}
	// update the title, description, and update time of the feed
	_, err = db.Query("INSERT INTO feeds VALUES(?, ?, ?, current_localtimestamp(), []) "+
		"ON CONFLICT DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, last_updated=EXCLUDED.last_updated",
		url, feed.Title, feed.Description)
	if err != nil {
		println(err.Error())
		panic("unable to update feed properties")
	}

	for i := 0; i < len(feed.Items); i++ {
		item := feed.Items[i]
		title := item.Title
		article := item.Link

		_, err = db.Query("INSERT OR IGNORE INTO articles VALUES (?, ?, ?, [], FALSE, FALSE)", article, title, item.PubDateParsed)
		if err != nil {
			println(err.Error())
			panic("unable to set insert article")
		}

		if len(item.Comments) == 0 {
			continue
		}

		_, err = db.Query("INSERT OR IGNORE INTO comments VALUES (?, ?, ?)", article, url, item.Comments)
		if err != nil {
			println(err.Error())
			panic("unable to set insert comments")
		}
	}
}

func archive_pages(db *sql.DB) {
	rows, err := db.Query("SELECT url FROM articles LEFT JOIN archive ON archive.article=articles.url WHERE archive.article IS NULL AND length(articles.tags) > 0")
	if err != nil {
		println(err.Error())
		panic("unable to set get articles to be archived")
	}
	for rows.Next() {
		var url string
		err = rows.Scan(&url)
		if err != nil {
			println(err.Error())
			panic("error scanning articles to be archived")
		}
		resp, err := http.Get(url)
		if err != nil || resp.StatusCode != 200 {
			println(err.Error())
			panic("error getting article")
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil || resp.StatusCode != 200 {
			println(err.Error())
			panic("error getting article")
		}
		fmt.Println(string(body))

		markdown, err := htmltomarkdown.ConvertString(string(body))
		if err != nil || resp.StatusCode != 200 {
			println(err.Error())
			panic("error converting to markdown")
		}
		fmt.Println(markdown)
	}
}
