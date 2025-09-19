package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "embed"

	_ "github.com/marcboeker/go-duckdb/v2"
)

type Article struct {
	Url        string
	EscapedUrl string
	Title      string
	Desc       string
	Date       string
	Comments   []Comments
	Tags       []string
}

type Articles struct {
	FavoriteTags []string
	Articles     []Article
}

type Comments struct {
	// The URL of the comments
	Url string
	// The URL of the feed the comments are from
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

// Page showing unread articles
var main_template *template.Template

// Page showing a list of feeds and input to add feeds
var feeds_template *template.Template

// Page showing an individual article
var article_template *template.Template

// Page showing the search menu
var search_template *template.Template

// Page showing an input to add bookmarks
var bookmark_template *template.Template

// API response for search results
var search_results_template *template.Template

// API response for an individual feed
var feed_template *template.Template

func main() {
	db = init_db()
	defer db.Close()
	var err error

	main_template, err = template.ParseFiles("templates/index.html", "templates/articles.html", "templates/header.html")
	if err != nil {
		panic(err.Error())
	}

	feeds_template, err = template.ParseFiles("templates/feeds.html", "templates/header.html")
	if err != nil {
		panic(err.Error())
	}

	article_template, err = template.ParseFiles("templates/article.html", "templates/article-component.html", "templates/header.html")
	if err != nil {
		panic(err.Error())
	}

	search_template, err = template.ParseFiles("templates/search.html", "templates/header.html")
	if err != nil {
		panic(err.Error())
	}

	bookmark_template, err = template.ParseFiles("templates/bookmark.html", "templates/header.html")
	if err != nil {
		panic(err.Error())
	}

	search_results_template, err = template.ParseFiles("templates/search-results.html", "templates/article-component.html")
	if err != nil {
		panic(err.Error())
	}

	feed_template, err = template.ParseFiles("templates/feed.html")
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

	mux.HandleFunc("POST /api/add_bookmark", add_bookmark)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/unread", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/unread", unread_handler)

	mux.HandleFunc("/feeds", feeds_handler)

	mux.HandleFunc("/article/{article}", article_handler)

	mux.HandleFunc("/search", search_handler)

	mux.HandleFunc("/bookmark", bookmark_handler)

	go func() {
		for {
			update_feeds(db)
			fmt.Println("updating all feeds")
			time.Sleep(60 * 60 * time.Second)
		}
	}()

	fmt.Println("server starting")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
