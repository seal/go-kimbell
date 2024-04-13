package models

type Post struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Url     string `json:"url"`
	Content string `json:"content"`
}
