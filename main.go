package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"time"

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

//go:embed static/*
var static embed.FS

//go:embed templates/*
var templates embed.FS

var db *sql.DB

// Page showing unread articles
var mainTemplate *template.Template

// Page showing a list of feeds and input to add feeds
var feedsTemplate *template.Template

// Page showing an individual article
var articleTemplate *template.Template

// Page showing the search menu
var search_template *template.Template

// Page showing an input to add bookmarks
var bookmark_template *template.Template

// API response for search results
var searchResultsTemplate *template.Template

// API response for an individual feed
var feed_template *template.Template

// component for a single article
var articleComponentTemplate *template.Template

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	var err error
	db, err = initDb()
	if err != nil {
		panic(err)
	}

	mainTemplate, err = template.ParseFS(templates, "templates/index.html", "templates/articles.html", "templates/header.html")
	if err != nil {
		panic(err)
	}

	feedsTemplate, err = template.ParseFS(templates, "templates/feeds.html", "templates/header.html")
	if err != nil {
		panic(err)
	}

	articleTemplate, err = template.ParseFS(templates, "templates/article.html", "templates/article-component.html", "templates/header.html")
	if err != nil {
		panic(err)
	}

	search_template, err = template.ParseFS(templates, "templates/search.html", "templates/header.html")
	if err != nil {
		panic(err)
	}

	bookmark_template, err = template.ParseFS(templates, "templates/bookmark.html", "templates/header.html")
	if err != nil {
		panic(err)
	}

	searchResultsTemplate, err = template.ParseFS(templates, "templates/search-results.html", "templates/article-component.html")
	if err != nil {
		panic(err)
	}

	feed_template, err = template.ParseFS(templates, "templates/feed.html")
	if err != nil {
		panic(err)
	}

	articleComponentTemplate, err = template.ParseFS(templates, "templates/article-component.html")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/mark_read", markRead)

	mux.HandleFunc("POST /api/remove_feed/", removeFeed)

	mux.HandleFunc("POST /api/add_tag/", addTag)

	mux.HandleFunc("POST /api/add_tag_mark_read", addTagMarkRead)

	mux.HandleFunc("POST /api/add_feed", addFeed)

	mux.HandleFunc("POST /api/search", searchQuery)

	mux.HandleFunc("POST /api/add_bookmark", addBookmark)

	mux.HandleFunc("POST /api/import_bookmarks", importBookmarks)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/unread", http.StatusTemporaryRedirect)
			return
		}
		file, err := static.ReadFile("static" + r.URL.Path)
		if err == nil {
			slog.DebugContext(r.Context(), "responding to", "path", r.URL.Path, "mime", mime.TypeByExtension(filepath.Ext(r.URL.Path)))
			w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(r.URL.Path)))
			w.Write(file)
			return
		}

		slog.Error("Page not found", "url", r.URL.String())

		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404"))
	})
	mux.HandleFunc("/unread", unreadHandler)

	mux.HandleFunc("/feeds", feedsHandler)

	mux.HandleFunc("/article/{article}", articleHandler)

	mux.HandleFunc("/search", searchHandler)

	mux.HandleFunc("/bookmark", bookmarkHandler)

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
