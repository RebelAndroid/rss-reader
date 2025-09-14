package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	article_list := query_articles_db(query)

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
