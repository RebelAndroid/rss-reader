package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed/rss"
)

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
	if tag[0] == '-' {
		remove_tag_db(url, tag[1:])
	} else {
		add_tag_db(url, tag)
	}
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
	start := time.Now()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
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
			add_feed_db(url + path)
			update_feed(db, url+path)
			feed_template.Execute(w, get_feed_db(url+path))
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

	article_list := query_articles_db(query)

	search_results_template.Execute(w, article_list)
}

func add_bookmark(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		panic(err.Error())
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		panic(err.Error())
	}

	url := parsed["url"][0]

	if !strings.HasPrefix(url, "https://") || !strings.HasPrefix(url, "http://") {
		url = "https://" + url
	}

	fmt.Println(url)

	resp, err := http.Get(url)
	if err != nil {
		panic(err.Error())
	}
	http_body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}

	regex, err := regexp.Compile("<title>.+</title>")
	if err != nil {
		panic(err.Error())
	}

	title := ""
	title_bytes := regex.Find(http_body)
	if title_bytes != nil {
		title = string(title_bytes[7 : len(title_bytes)-8])
	}
	fmt.Println(title)

	add_bookmark_db(url, title)

	w.Write([]byte("Bookmark added successfully"))
}

func unread_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	article_list := unread_articles_db(10)

	articles := Articles{
		FavoriteTags: []string{"later", "favorite", "reference", "archive"},
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

func bookmark_handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		println("unexpected method")
		return
	}

	err := bookmark_template.Execute(w, nil)
	if err != nil {
		panic(err.Error())
	}
}
