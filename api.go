package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed/rss"
)

func removeFeed(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	url := parsed["url"][0]
	removeFeedDb(url)
}

func markRead(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	url := parsed["url"][0]
	markReadDb(url)
}

func addTag(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	url := parsed["url"][0]
	tag := parsed["tag"][0]
	if tag[0] == '-' {
		removeTagDb(url, tag[1:])
	} else {
		addTagDb(url, tag)
	}

	article := getArticleDb(url)

	articleComponentTemplate.Execute(w, article)
}

func addTagMarkRead(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	url := parsed["url"][0]
	tag := parsed["tag"][0]
	addTagDb(url, tag)
	markReadDb(url)
}

func addFeed(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	url := parsed["url"][0]

	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	// the different paths to look for a feed in
	// TODO: parallelize this
	paths := []string{"", "/rss", "/index.xml", "/feed"}

	for _, path := range paths {
		fmt.Println("trying path: " + string(url+path))
		resp, err := http.Get(url + path)
		if err == nil && resp.StatusCode == 200 {
			fmt.Println("success path: " + string(url+path))
			parser := rss.Parser{}
			_, err = parser.Parse(resp.Body)
			if err != nil {
				fmt.Println("unable to read feed body")
				continue
			}
			addFeedDb(url + path)
			update_feed(db, url+path)
			feed_template.Execute(w, getFeedDb(url+path))
			break
		} else if err != nil {
			fmt.Printf("got err: %s status code: %d\n", err.Error(), resp.StatusCode)
		} else {
			fmt.Printf("got status code: %d\n", resp.StatusCode)
		}
	}
	end := time.Now()
	fmt.Println("ran /add/feed in " + end.Sub(start).String())
}

func searchQuery(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	query := parsed["query"][0]

	articleList := queryArticlesDb(query)

	searchResultsTemplate.Execute(w, articleList)
}

func addBookmark(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	url := parsed["url"][0]

	if !strings.HasPrefix(url, "https://") || !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	httpBody, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	regex, err := regexp.Compile("<title>.+</title>")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	title := ""
	titleBytes := regex.Find(httpBody)
	if titleBytes != nil {
		title = string(titleBytes[7 : len(titleBytes)-8])
	}
	fmt.Println(title)

	addBookmarkDb(url, title)

	w.Write([]byte("Bookmark added successfully"))
}

func unreadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	articleList := unreadArticlesDb(10)

	articles := Articles{
		FavoriteTags: []string{"later", "favorite", "reference", "archive"},
		Articles:     articleList,
	}

	err := mainTemplate.Execute(w, articles)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func feedsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	feeds := feedsDb()

	err := feedsTemplate.Execute(w, feeds)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
	article_url := r.PathValue("article")
	if article_url == "" {
		panic("couldn't get article out of path")
	}
	article := getArticleDb(article_url)

	articleTemplate.Execute(w, article)
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	articleList := readArticlesDb(20)

	err := search_template.Execute(w, articleList)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func bookmarkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	err := bookmark_template.Execute(w, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

type Bookmark struct {
	URI   string `json:"uri"`
	Title string `json:"title"`
}

type BookmarkFolder struct {
	Children []Bookmark `json:"children"`
}

type Bookmarks struct {
	Guid     string           `json:"guid"`
	Children []BookmarkFolder `json:"children"`
}

func importBookmarks(w http.ResponseWriter, r *http.Request) {
	slog.Debug("importing bookmarks")
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		slog.ErrorContext(r.Context(), "request lacks Content-Type header", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	mr := multipart.NewReader(r.Body, params["boundary"])
	// 4MiB should be enough TODO: make this configurable
	form, err := mr.ReadForm(4 * 1024 * 1024)
	if err != nil {
		slog.ErrorContext(r.Context(), "error parsing form", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	fileHeaders := form.File["file"]
	fmt.Println(len(fileHeaders))
	file, err := fileHeaders[0].Open()
	if err != nil {
		slog.ErrorContext(r.Context(), "error opening form file", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	bytes, err := io.ReadAll(file)
	if err != nil {
		slog.ErrorContext(r.Context(), "error opening form file", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	var bookmarks Bookmarks
	err = json.Unmarshal(bytes, &bookmarks)
	if err != nil {
		slog.Error("error opening form file", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	slog.Debug("parsed bookmarks", "folders", len(bookmarks.Children), "guid", bookmarks.Guid)

	for _, folder := range bookmarks.Children {
		slog.Debug("parsed bookmarks folder", "bookmarks", len(folder.Children))
		for _, bookmark := range folder.Children {
			slog.Debug("got bookmark", "bookmark", bookmark)
			if strings.HasPrefix(bookmark.URI, "http") {
				addBookmarkDb(bookmark.URI, bookmark.Title)
			}
		}
	}

}
