package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	_ "embed"

	_ "github.com/marcboeker/go-duckdb/v2"
	"github.com/mmcdole/gofeed/rss"
)

type article struct {
	Url        string
	EscapedUrl string
	Title      string
	Desc       string
	Date       string
	Comments   []comments
	Tags       []string
}

type articles struct {
	FavoriteTags []string
	Articles     []article
}

type comments struct {
	Url  string
	Feed string
}

type feed struct {
	// The URL of the website this feed is for (ie mbund.dev)
	SiteUrl string
	// The URL of the actual feed (ie mbund.dev/index.xml)
	FeedUrl     string
	Title       string
	Description string
}

//go:embed static/index.css
var index_css string

//go:embed static/index.js
var index_js string

//go:embed static/preflight.css
var preflight_css string

//go:embed static/htmx.min.js
var htmx_min_js string

var db *sql.DB

// var articles_template *template.Template
var main_template *template.Template

var feeds_template *template.Template

var article_template *template.Template

var search_template *template.Template

var search_results_template *template.Template

func main() {
	db = init_db()
	defer db.Close()
	var err error

	main_template, err = template.ParseFiles("templates/index.html", "templates/articles.html")
	if err != nil {
		panic(err.Error())
	}

	feeds_template, err = template.ParseFiles("templates/feeds.html")
	if err != nil {
		panic(err.Error())
	}

	article_template, err = template.ParseFiles("templates/article.html")
	if err != nil {
		panic(err.Error())
	}

	search_template, err = template.ParseFiles("templates/search.html")
	if err != nil {
		panic(err.Error())
	}

	search_results_template, err = template.ParseFiles("templates/search-results.html")
	if err != nil {
		panic(err.Error())
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /index.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		fmt.Fprint(w, index_css)
	})

	mux.HandleFunc("GET /preflight.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/css")
		fmt.Fprint(w, preflight_css)
	})

	mux.HandleFunc("GET /index.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/javascript")
		fmt.Fprint(w, index_js)
	})

	mux.HandleFunc("GET /htmx.min.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/javascript")
		fmt.Fprint(w, htmx_min_js)
	})

	mux.HandleFunc("POST /api/mark_read", mark_read)

	mux.HandleFunc("POST /api/remove_feed/", remove_feed)

	mux.HandleFunc("POST /api/add_tag/", add_tag)

	mux.HandleFunc("POST /api/add_tag_mark_read", add_tag_mark_read)

	mux.HandleFunc("POST /api/add_feed", add_feed)

	mux.HandleFunc("POST /api/search", search_query)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/unread", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/unread", unread_handler)

	mux.HandleFunc("/feeds", feeds_handler)

	mux.HandleFunc("/article/{article}", article_handler)

	mux.HandleFunc("/search", search_handler)

	update_feed(db, "https://lobste.rs/rss")
	update_feed(db, "https://news.ycombinator.com/rss")
	update_feed(db, "https://mbund.dev/index.xml")

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func remove_feed(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	url := parsed["url"][0]
	remove_feed_db(url)
}

func mark_read(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	url := parsed["url"][0]
	mark_read_db(url)
}

func add_tag(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	url := parsed["url"][0]
	tag := parsed["tag"][0]
	add_tag_db(url, tag)
}

func add_tag_mark_read(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	url := parsed["url"][0]
	tag := parsed["tag"][0]
	add_tag_db(url, tag)
	mark_read_db(url)
}

func add_feed(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	url := parsed["url"][0]
	add_feed_db(url)
	update_feed(db, url)
}

func search_query(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	query := parsed["query"][0]
	fmt.Println(query)

	article_list := read_articles_db(5)

	search_results_template.Execute(w, article_list)
}

func unread_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	article_list := unread_articles_db(10)

	articles := articles{
		FavoriteTags: []string{"Read Later", "Favorite", "Reference"},
		Articles:     article_list,
	}

	err := main_template.Execute(w, articles)
	if err != nil {
		panic(err.Error())
	}
}

func feeds_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	feeds := feeds_db()

	err := feeds_template.Execute(w, feeds)
	if err != nil {
		panic(err.Error())
	}
}

func article_handler(w http.ResponseWriter, r *http.Request) {
	article_url := r.PathValue("article")
	if article_url == "" {
		panic("couldn't get article out of path")
	}
	article := get_article_db(article_url)

	article_template.Execute(w, article)
}

func search_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	article_list := read_articles_db(20)

	err := search_template.Execute(w, article_list)
	if err != nil {
		panic(err.Error())
	}
}

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
	// 304 is "not modified", if this is the status, nothing has changed so we don't need to update it
	// we might need to check if the body is empty as well, but I don't care
	if resp.StatusCode == 304 {
		return
	}

	defer resp.Body.Close()

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
		date, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			panic(err.Error())
		}

		_, err = db.Query("INSERT OR IGNORE INTO articles VALUES (?, ?, ?, ?, [], FALSE)", article, title, "description", date)
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
