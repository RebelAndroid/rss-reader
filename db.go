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

func initDb() (*sql.DB, error) {
	db, err := sql.Open("duckdb", "data/data.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS feeds" +
		"(url STRING PRIMARY KEY, " +
		"title STRING NOT NULL, " +
		"description STRING NOT NULL, " +
		"last_updated TIMESTAMP," +
		"tags STRING[])")
	if err != nil {
		return nil, fmt.Errorf("failed to create table feeds: %v", err)
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS tags " +
		"(name STRING NOT NULL PRIMARY KEY," +
		"favorite BOOL NOT NULL)")
	if err != nil {
		return nil, fmt.Errorf("failed to create table tags: %v", err)
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS articles " +
		"(url STRING NOT NULL PRIMARY KEY, " +
		"title STRING NOT NULL, " +
		"description STRING, " +
		"pubdate TIMESTAMP, " +
		"tags STRING[]," +
		"read BOOL)")
	if err != nil {
		return nil, fmt.Errorf("failed to create table articles: %v", err)
	}

	_, err = db.Query("CREATE TABLE IF NOT EXISTS comments" +
		"(article STRING NOT NULL," +
		"feed STRING NOT NULL," +
		"comments STRING NOT NULL PRIMARY KEY)")

	if err != nil {
		return nil, fmt.Errorf("failed to create table comments: %v", err)
	}

	return db, nil
}

func removeFeedDb(url string) error {
	_, err := db.Exec("DELETE FROM comments WHERE feed=?", url)
	if err != nil {
		return fmt.Errorf("failed to delete feed comments: %v", err)
	}
	res, err := db.Exec("DELETE FROM feeds WHERE url=?", url)
	if err != nil {
		return fmt.Errorf("failed to delete feed: %v", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("attempted to delete nonexistent feed: %s", url)
	}
	return nil
}

func markReadDb(url string) {
	res, err := db.Exec("UPDATE articles SET read=true WHERE url=?", url)
	if err != nil {
		panic(err)
	}
	i, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}
	if i != 1 {
		println("got invalid url: " + url)
	}
}

func unreadArticlesDb(limit int) []Article {
	articleRows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE read=false ORDER BY pubdate DESC LIMIT ?", limit)
	if err != nil {
		panic(err)
	}

	var articleList []Article
	for articleRows.Next() {
		var article Article
		_ = articleRows.Scan(&article.Url, &article.Title, &article.Desc, &article.Date)
		article.EscapedUrl = url.QueryEscape(article.Url)

		date, err := time.Parse(time.RFC3339, article.Date)
		if err != nil {
			panic(err)
		}
		article.Date = date.Format(time.RFC1123)

		comment_rows, err := db.Query("SELECT title, comments FROM comments JOIN feeds ON feed=url WHERE article=?", article.Url)
		if err != nil {
			panic(err)
		}
		var commentsArray []Comments
		for comment_rows.Next() {
			var comment Comments
			_ = comment_rows.Scan(&comment.Feed, &comment.Url)
			commentsArray = append(commentsArray, comment)
		}
		article.Comments = commentsArray

		articleList = append(articleList, article)
	}
	return articleList
}

func readArticlesDb(limit int) []Article {
	articleRows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE read=true ORDER BY pubdate DESC LIMIT ?", limit)
	if err != nil {
		panic(err)
	}

	var articleList []Article
	for articleRows.Next() {
		var article Article
		_ = articleRows.Scan(&article.Url, &article.Title, &article.Desc, &article.Date)
		article.EscapedUrl = url.QueryEscape(article.Url)

		date, err := time.Parse(time.RFC3339, article.Date)
		if err != nil {
			panic(err)
		}
		article.Date = date.Format(time.RFC1123)

		commentRows, err := db.Query("SELECT title, comments FROM comments JOIN feeds ON feed=url WHERE article=?", article.Url)
		if err != nil {
			panic(err)
		}
		var commentsArray []Comments
		for commentRows.Next() {
			var comment Comments
			_ = commentRows.Scan(&comment.Feed, &comment.Url)
			commentsArray = append(commentsArray, comment)
		}
		article.Comments = commentsArray

		articleList = append(articleList, article)
	}
	return articleList
}

func queryArticlesDb(query string) []Article {
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

	articleRows, err := db.Query("SELECT url, title, description,pubdate FROM articles WHERE " + condition)
	if err != nil {
		panic(err)
	}

	var articleList []Article
	for articleRows.Next() {
		var article Article
		_ = articleRows.Scan(&article.Url, &article.Title, &article.Desc, &article.Date)
		article.EscapedUrl = url.QueryEscape(article.Url)

		date, err := time.Parse(time.RFC3339, article.Date)
		if err != nil {
			panic(err)
		}
		article.Date = date.Format(time.RFC1123)

		commentRows, err := db.Query("SELECT title, comments FROM comments JOIN feeds ON feed=url WHERE article=?", article.Url)
		if err != nil {
			panic(err)
		}
		var comments_array []Comments
		for commentRows.Next() {
			var comment Comments
			_ = commentRows.Scan(&comment.Feed, &comment.Url)
			comments_array = append(comments_array, comment)
		}
		article.Comments = comments_array

		articleList = append(articleList, article)
	}
	return articleList
}

func feedsDb() []feed {
	feed_rows, err := db.Query("SELECT url, title, description FROM feeds")
	if err != nil {
		panic(err)
	}

	var feeds []feed
	for feed_rows.Next() {
		var feed feed
		_ = feed_rows.Scan(&feed.FeedUrl, &feed.Title, &feed.Description)
		url, err := url.Parse(feed.FeedUrl)
		if err != nil {
			panic(err)
		}
		feed.SiteUrl = "https://" + url.Host
		feeds = append(feeds, feed)
	}

	return feeds
}

func getFeedDb(requested_url string) feed {
	start := time.Now()
	feed_row := db.QueryRow("SELECT url, title, description FROM feeds WHERE url=?", requested_url)

	var feed feed
	err := feed_row.Scan(&feed.FeedUrl, &feed.Title, &feed.Description)
	if err != nil {
		panic(err)
	}
	feedUrl, err := url.Parse(feed.FeedUrl)
	if err != nil {
		panic(err)
	}
	feed.SiteUrl = "https://" + feedUrl.Host
	end := time.Now()
	fmt.Println(end.Sub(start))

	return feed
}

func addTagDb(url string, tag string) {
	_, err := db.Query("UPDATE articles SET tags=list_distinct(list_append(tags, ?)) WHERE url=?", tag, url)
	if err != nil {
		panic(err)
	}
}

func removeTagDb(url string, tag string) {
	_, err := db.Query("UPDATE articles SET tags=list_filter(tags, lambda x: x != ?) WHERE url=?", tag, url)
	if err != nil {
		panic(err)
	}
}

func addFeedDb(url string) {
	_, err := db.Query("INSERT INTO feeds VALUES(?, '', '', NULL, [])", url)
	if err != nil {
		panic(err)
	}
}

func getArticleDb(article_url string) Article {
	var article Article
	// I don't think there is any reason for the feed names and comment links to match up to each other
	// TODO: fix that
	row := db.QueryRow("SELECT list_filter(list(comments), lambda x: x != NULL), list_filter(list(feeds.title), lambda x: x != NULL), ANY_VALUE(articles.url), ANY_VALUE(articles.title), ANY_VALUE(articles.description), ANY_VALUE(pubdate), ANY_VALUE(articles.tags) FROM articles LEFT JOIN comments ON articles.url=comments.article LEFT JOIN feeds ON comments.feed=feeds.url WHERE articles.url=? GROUP BY articles.url;", article_url)
	var tagsArr duckdb.Composite[[]string]
	var commentsArr duckdb.Composite[[]string]
	var feedCommentsArr duckdb.Composite[[]string]
	err := row.Scan(&commentsArr, &feedCommentsArr, &article.Url, &article.Title, &article.Desc, &article.Date, &tagsArr)
	if err != nil {
		panic(err)
	}
	article.Tags = tagsArr.Get()

	comments_str := commentsArr.Get()

	feeds := feedCommentsArr.Get()

	for i, comment := range comments_str {
		feed := feeds[i]
		article.Comments = append(article.Comments, Comments{comment, feed})
	}

	article.EscapedUrl = url.QueryEscape(article.Url)

	return article
}

func addBookmarkDb(url string, title string) {
	_, err := db.Query("INSERT OR IGNORE INTO articles VALUES (?, ?, ?, ?, ['bookmark'], FALSE)", url, title, "description", time.Now().Format(time.RFC3339))
	if err != nil {
		panic(err)
	}
}
