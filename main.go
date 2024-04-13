package main

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/seal/go-kimbell/pkg/handlers"
	"github.com/seal/go-kimbell/pkg/utils"
)

// Test via
/*
	curl -X POST \
	 -H "Content-Type: application/json" \
	 -H "Authorization: x" \
	 -d '{
	   "date": "2024-01-08",
	   "title": "newerMy New xxxxx Post",
	   "url": "my-new-xxxxx-post",
	   "content": "# My New xxxxxx Post\n\nThis is the content of my new blog post."
	 }' \
	 http://localhost:8000/api/new
*/

func main() {

	err := utils.GeneratePosts()
	if err != nil {
		slog.Error("Failed to generate posts", err)
		return
	}
	file, err := os.OpenFile("log.json", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Failed to open log file", err)
		os.Exit(1)
	}
	defer file.Close()
	// Set default logger to a new json one where the writer is stdOUT and the file
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.MultiWriter(file, os.Stdout), &slog.HandlerOptions{
		Level: slog.LevelInfo})))
	r := mux.NewRouter()
	r.HandleFunc("/api/new", handlers.NewPost).Methods("POST")
	r.HandleFunc("/", handlers.IndexHandler)
	r.HandleFunc("/posts/{url}", handlers.GetPost).Methods("GET")
	r.HandleFunc("/index.html", handlers.IndexHandler)
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.ico")
	})
	r.HandleFunc("/tomorrow-night.css", func(w http.ResponseWriter, r *http.Request) {
		css, err := os.ReadFile("tomorrow-night.css")
		if err != nil {
			http.Error(w, "Error reading css", http.StatusInternalServerError)
			slog.Error(err.Error())
		}
		w.Header().Set("Content-Type", "text/css")
		w.Write(css)
	})
	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:3000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	slog.Info("Starting server")
	log.Fatal(srv.ListenAndServe())

}
