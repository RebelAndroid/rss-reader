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
	db, err := sql.Open("duckdb", "data/data.db")
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

func unread_articles_db(limit int) []Article {
	article_rows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE read=false ORDER BY pubdate DESC LIMIT ?", limit)
	if err != nil {
		panic(err.Error())
	}

	var article_list []Article
	for article_rows.Next() {
		var article Article
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
		var comments_array []Comments
		for comment_rows.Next() {
			var comment Comments
			_ = comment_rows.Scan(&comment.Feed, &comment.Url)
			comments_array = append(comments_array, comment)
		}
		article.Comments = comments_array

		article_list = append(article_list, article)
	}
	return article_list
}

func read_articles_db(limit int) []Article {
	article_rows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE read=true ORDER BY pubdate DESC LIMIT ?", limit)
	if err != nil {
		panic(err.Error())
	}

	var article_list []Article
	for article_rows.Next() {
		var article Article
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
		var comments_array []Comments
		for comment_rows.Next() {
			var comment Comments
			_ = comment_rows.Scan(&comment.Feed, &comment.Url)
			comments_array = append(comments_array, comment)
		}
		article.Comments = comments_array

		article_list = append(article_list, article)
	}
	return article_list
}

func query_articles_db(query string) []Article {
	r := regexp.MustCompile(`^#?([a-z]|[A-Z]|[0-9])+$`)

	parts := strings.Split(query, " ")
	condition := ""
	for _, word := range parts {
		fmt.Println(word)
		if !r.MatchString(word) {
			fmt.Printf("Bad search query: %s, problem: %s\n", query, word)
			return []Article{}
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

	var article_list []Article
	for article_rows.Next() {
		var article Article
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
		var comments_array []Comments
		for comment_rows.Next() {
			var comment Comments
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

func get_feed_db(requested_url string) feed {
	start := time.Now()
	feed_row := db.QueryRow("SELECT url, title, description FROM feeds WHERE url=?", requested_url)

	var feed feed
	err := feed_row.Scan(&feed.FeedUrl, &feed.Title, &feed.Description)
	if err != nil {
		panic(err.Error())
	}
	feed_url, err := url.Parse(feed.FeedUrl)
	if err != nil {
		panic(err.Error())
	}
	feed.SiteUrl = "https://" + feed_url.Host
	end := time.Now()
	fmt.Println(end.Sub(start))

	return feed
}

func add_tag_db(url string, tag string) {
	_, err := db.Query("UPDATE articles SET tags=list_distinct(list_append(tags, ?)) WHERE url=?", tag, url)
	if err != nil {
		panic(err.Error())
	}
}

func remove_tag_db(url string, tag string) {
	_, err := db.Query("UPDATE articles SET tags=list_filter(tags, lambda x: x != ?) WHERE url=?", tag, url)
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

func get_article_db(article_url string) Article {
	var article Article
	// I don't think there is any reason for the feed names and comment links to match up to each other
	// TODO: fix that
	row := db.QueryRow("SELECT list_filter(list(comments), lambda x: x != NULL), list_filter(list(feeds.title), lambda x: x != NULL), ANY_VALUE(articles.url), ANY_VALUE(articles.title), ANY_VALUE(articles.description), ANY_VALUE(pubdate), ANY_VALUE(articles.tags) FROM articles LEFT JOIN comments ON articles.url=comments.article LEFT JOIN feeds ON comments.feed=feeds.url WHERE articles.url=? GROUP BY articles.url;", article_url)
	var tags_arr duckdb.Composite[[]string]
	var comments_arr duckdb.Composite[[]string]
	var feed_comments_arr duckdb.Composite[[]string]
	err := row.Scan(&comments_arr, &feed_comments_arr, &article.Url, &article.Title, &article.Desc, &article.Date, &tags_arr)
	if err != nil {
		panic(err.Error())
	}
	article.Tags = tags_arr.Get()

	comments_str := comments_arr.Get()

	feeds := feed_comments_arr.Get()

	for i, comment := range comments_str {
		feed := feeds[i]
		article.Comments = append(article.Comments, Comments{comment, feed})
	}

	article.EscapedUrl = url.QueryEscape(article.Url)

	return article
}

func add_bookmark_db(url string, title string) {
	_, err := db.Query("INSERT OR IGNORE INTO articles VALUES (?, ?, ?, ?, ['bookmark'], FALSE)", url, title, "description", time.Now().Format(time.RFC3339))
	if err != nil {
		panic(err.Error())
	}
}
