package main

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/marcboeker/go-duckdb/v2"
)

func init_db() *sql.DB {
	db, err := sql.Open("duckdb", "./data.db")
	if err != nil {
		println(err.Error())
		panic("unable to open database")
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS feeds" +
		"(url STRING PRIMARY KEY, " +
		"title STRING NOT NULL, " +
		"description STRING NOT NULL, " +
		"last_updated TIMESTAMP," +
		"tags STRING[])")
	if err != nil {
		println(err.Error())
		panic("unable to create feeds table")
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS tags " +
		"(name STRING NOT NULL PRIMARY KEY," +
		"favorite BOOL NOT NULL)")
	if err != nil {
		println(err.Error())
		panic("unable to create tags table")
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS articles " +
		"(url STRING NOT NULL PRIMARY KEY, " +
		"title STRING NOT NULL, " +
		"description STRING, " +
		"pubdate TIMESTAMP, " +
		"tags STRING[]," +
		"read BOOL)")
	if err != nil {
		println(err.Error())
		panic("unable to create articles table")
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS comments" +
		"(article STRING NOT NULL," +
		"feed STRING NOT NULL," +
		"comments STRING NOT NULL PRIMARY KEY)")

	if err != nil {
		println(err.Error())
		panic("unable to create comments table")
	}

	return db
}

func remove_feed_db(url string) {
	_, err := db.Exec("DELETE FROM comments WHERE feed=?", url)
	if err != nil {
		panic(err.Error())
	}
	res, err := db.Exec("DELETE FROM feeds WHERE url=?", url)
	if err != nil {
		panic(err.Error())
	}
	rows, err := res.RowsAffected()
	if err != nil {
		panic(err.Error())
	}
	if rows == 0 {
		panic("attempted to delete non existent feed")
	}
}

func mark_read_db(url string) {
	res, err := db.Exec("UPDATE articles SET read=true WHERE url=?", url)
	if err != nil {
		panic(err.Error())
	}
	i, err := res.RowsAffected()
	if err != nil {
		panic(err.Error())
	}
	if i != 1 {
		println("got invalid url: " + url)
	}
}

func unread_articles_db(limit int) []article {
	article_rows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE read=false ORDER BY pubdate DESC LIMIT ?", limit)
	if err != nil {
		panic(err.Error())
	}

	var article_list []article
	for article_rows.Next() {
		var article article
		_ = article_rows.Scan(&article.Url, &article.Title, &article.Desc, &article.Date)
		article.EscapedUrl = url.QueryEscape(article.Url)

		date, err := time.Parse(time.RFC3339, article.Date)
		if err != nil {
			panic(err.Error())
		}
		article.Date = date.Format(time.RFC1123)

		comment_rows, err := db.Query("SELECT title, comments FROM comments JOIN feeds ON feed=url WHERE article=?", article.Url)
		if err != nil {
			panic(err.Error())
		}
		var comments_array []comments
		for comment_rows.Next() {
			var comment comments
			_ = comment_rows.Scan(&comment.Feed, &comment.Url)
			comments_array = append(comments_array, comment)
		}
		article.Comments = comments_array

		article_list = append(article_list, article)
	}
	return article_list
}

func read_articles_db(limit int) []article {
	article_rows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE read=true ORDER BY pubdate DESC LIMIT ?", limit)
	if err != nil {
		panic(err.Error())
	}

	var article_list []article
	for article_rows.Next() {
		var article article
		_ = article_rows.Scan(&article.Url, &article.Title, &article.Desc, &article.Date)
		article.EscapedUrl = url.QueryEscape(article.Url)

		date, err := time.Parse(time.RFC3339, article.Date)
		if err != nil {
			panic(err.Error())
		}
		article.Date = date.Format(time.RFC1123)

		comment_rows, err := db.Query("SELECT title, comments FROM comments JOIN feeds ON feed=url WHERE article=?", article.Url)
		if err != nil {
			panic(err.Error())
		}
		var comments_array []comments
		for comment_rows.Next() {
			var comment comments
			_ = comment_rows.Scan(&comment.Feed, &comment.Url)
			comments_array = append(comments_array, comment)
		}
		article.Comments = comments_array

		article_list = append(article_list, article)
	}
	return article_list
}

func query_articles_db(query string) []article {
	r := regexp.MustCompile(`^#?([a-z]|[A-Z]|[0-9])+$`)

	parts := strings.Split(query, " ")
	condition := ""
	for _, word := range parts {
		fmt.Println(word)
		if !r.MatchString(word) {
			fmt.Printf("Bad search query: %s, problem: %s\n", query, word)
			return []article{}
		}
		if strings.HasPrefix(word, "#") {
			condition = condition + "list_contains(tags, '" + word[1:] + "') AND "
		} else {
			condition = condition + "title ILIKE '%" + word + "%' AND "
		}
	}

	condition = condition + "true"

	fmt.Println("condition: " + condition)

	article_rows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE " + condition)
	if err != nil {
		panic(err.Error())
	}

	var article_list []article
	for article_rows.Next() {
		var article article
		_ = article_rows.Scan(&article.Url, &article.Title, &article.Desc, &article.Date)
		article.EscapedUrl = url.QueryEscape(article.Url)

		date, err := time.Parse(time.RFC3339, article.Date)
		if err != nil {
			panic(err.Error())
		}
		article.Date = date.Format(time.RFC1123)

		comment_rows, err := db.Query("SELECT title, comments FROM comments JOIN feeds ON feed=url WHERE article=?", article.Url)
		if err != nil {
			panic(err.Error())
		}
		var comments_array []comments
		for comment_rows.Next() {
			var comment comments
			_ = comment_rows.Scan(&comment.Feed, &comment.Url)
			comments_array = append(comments_array, comment)
		}
		article.Comments = comments_array

		article_list = append(article_list, article)
	}
	return article_list
}

func feeds_db() []feed {
	feed_rows, err := db.Query("SELECT url, title, description FROM feeds")
	if err != nil {
		panic(err.Error())
	}

	var feeds []feed
	for feed_rows.Next() {
		var feed feed
		_ = feed_rows.Scan(&feed.FeedUrl, &feed.Title, &feed.Description)
		url, err := url.Parse(feed.FeedUrl)
		if err != nil {
			panic(err.Error())
		}
		feed.SiteUrl = "https://" + url.Host
		feeds = append(feeds, feed)
	}

	return feeds
}

func add_tag_db(url string, tag string) {
	_, err := db.Query("UPDATE articles SET tags=list_distinct(list_append(tags, ?)) WHERE url=?", tag, url)
	if err != nil {
		panic(err.Error())
	}
}

func add_feed_db(url string) {
	_, err := db.Query("INSERT INTO feeds VALUES(?, '', '', NULL, [])", url)
	if err != nil {
		panic(err.Error())
	}
}

func get_article_db(article_url string) article {
	var article article
	row := db.QueryRow("SELECT url, title, description, pubdate, tags FROM articles WHERE url=?", article_url)
	var arr duckdb.Composite[[]string]
	err := row.Scan(&article.Url, &article.Title, &article.Desc, &article.Date, &arr)
	article.Tags = arr.Get()
	if err != nil {
		panic(err.Error())
	}
	article.EscapedUrl = url.QueryEscape(article.Url)

	return article
}
