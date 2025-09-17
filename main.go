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

var feed_template *template.Template

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

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/unread", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/unread", unread_handler)

	mux.HandleFunc("/feeds", feeds_handler)

	mux.HandleFunc("/article/{article}", article_handler)

	mux.HandleFunc("/search", search_handler)

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
