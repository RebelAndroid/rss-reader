package main

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"strings"
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
		slog.Error("unable to get feed", "feed", url, "error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return
	}

	fp := rss.Parser{}
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		slog.Error("unable to parse feed", "feed", url, "error", err.Error())
	}
	// update the title, description, and update time of the feed
	_, err = db.Query("INSERT INTO feeds VALUES(?, ?, ?, current_localtimestamp(), []) "+
		"ON CONFLICT DO UPDATE SET title=EXCLUDED.title, description=EXCLUDED.description, last_updated=EXCLUDED.last_updated",
		url, feed.Title, feed.Description)
	if err != nil {
		slog.Error("unable to update feed properties", "feed", url, "error", err.Error())
		return
	}

	for i := 0; i < len(feed.Items); i++ {
		item := feed.Items[i]
		title := item.Title

		article := Article{
			Url:        item.Link,
			EscapedUrl: "",
			Title:      title,
			Date:       item.PubDateParsed.Format(time.RFC3339),
			Comments:   []Comments{},
			Tags:       []string{},
		}

		err = addArticleDb(article)
		if err != nil {
			slog.Error("unable to add article", "feed", url, "article", item.Link, "error", err.Error())
			continue
		}

		if len(item.Comments) == 0 {
			continue
		}

		_, err = db.Query("INSERT OR IGNORE INTO comments VALUES (?, ?, ?)", item.Link, url, item.Comments)
		if err != nil {
			slog.Error("unable to add comments", "feed", url, "article", item.Link, "comments", item.Comments, "error", err.Error())
			continue
		}
	}
}

func archive_pages(db *sql.DB) {
	rows, err := db.Query("SELECT url FROM articles WHERE archive IS NULL AND length(articles.tags) > 0 AND NOT dead_link")
	if err != nil {
		println(err.Error())
		panic("unable to get articles to be archived")
	}
	for rows.Next() {
		var url string
		err = rows.Scan(&url)
		if err != nil {
			println(err.Error())
			panic("error scanning articles to be archived")
		}
		resp, err := http.Get(url)
		if err != nil {
			slog.Error("unable to get article", "url", url, "error", err)
			_, err = db.Exec("UPDATE articles SET dead_link=true WHERE url=?", url)
			if err != nil {
				slog.Error("unable to mark dead link in db", "error", err, "url", url)
				continue
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {

			slog.Error("unable to get article", "url", url, "status", resp.StatusCode)
			_, err = db.Exec("UPDATE articles SET dead_link=true WHERE url=?", url)
			if err != nil {
				slog.Error("unable to mark dead link in db", "error", err, "url", url)
				continue
			}
			continue
		}

		content_type := resp.Header.Get("content-type")
		if strings.Contains(content_type, "text/html") {
			slog.Error("link is not HTML, not archiving", "url", url, "content-type", content_type)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("unable to read body of article", "url", url)
			continue
		}

		markdown, err := htmltomarkdown.ConvertString(string(body))
		if err != nil {
			slog.Error("unable to convert to markdown", "error", err, "url", url)
			continue
		}

		_, err = db.Exec("UPDATE articles SET archive=? WHERE url=?", markdown, url)
		if err != nil {
			slog.Error("unable to add archive to db", "error", err, "url", url)
			continue
		}
		_, err = db.Exec("UPDATE articles SET dead_link=FALSE WHERE url=?", url)
		if err != nil {
			slog.Error("unable to add archive to db", "error", err, "url", url)
			continue
		}
	}
}
