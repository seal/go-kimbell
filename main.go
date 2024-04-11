package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func newPost(w http.ResponseWriter, r *http.Request) {
	uid, err := uuid.NewV6()
	if err != nil {
		http.Error(w, "Failed to generate request ID", http.StatusInternalServerError)
		slog.Error("Failed to generate request ID", err)
		return
	}
	reqID := uid.String()
	slog.Info("New post request received", slog.String("request-id", reqID))
	apiKey := strings.TrimSpace(r.Header.Get("Authorization"))
	if apiKey != envVariable("APIKEY") || apiKey == "" {
		http.Error(w, "Unauthorized: Invalid API key", http.StatusUnauthorized)
		slog.Error("Unauthorized request", slog.String("request-id", reqID), slog.String("key", apiKey))
		return
	}
	var post Post
	err = json.NewDecoder(r.Body).Decode(&post)
	if err != nil {
		http.Error(w, "Bad Request: Invalid JSON payload", http.StatusBadRequest)
		slog.Error("Failed to parse JSON payload", err, slog.String("request-id", reqID))
		return
	}
	defer r.Body.Close()

	if post.Date == "" || post.Title == "" || post.Url == "" || post.Content == "" {
		http.Error(w, "Bad Request: Missing required fields", http.StatusBadRequest)
		slog.Error("Missing required fields", slog.String("request-id", reqID))
		return
	}

	err = os.MkdirAll("posts", os.ModePerm)
	if err != nil {
		http.Error(w, "Internal Server Error: Failed to create posts directory", http.StatusInternalServerError)
		slog.Error("Failed to create posts directory", err, slog.String("request-id", reqID))
		return
	}

	jsonData, err := json.Marshal(post)
	if err != nil {
		http.Error(w, "Internal Server Error: Failed to marshal JSON data", http.StatusInternalServerError)
		slog.Error("Failed to marshal JSON data", err, slog.String("request-id", reqID))
		return
	}
	jsonFileName := filepath.Join("posts", post.Url+".json")
	err = os.WriteFile(jsonFileName, jsonData, 0644)
	if err != nil {
		http.Error(w, "Internal Server Error: Failed to write JSON file", http.StatusInternalServerError)
		slog.Error("Failed to write JSON file", err, slog.String("request-id", reqID))
		return
	}

	markdownFileName := filepath.Join("posts", post.Url+".md")
	err = os.WriteFile(markdownFileName, []byte(post.Content), 0644)
	if err != nil {
		http.Error(w, "Internal Server Error: Failed to write Markdown file", http.StatusInternalServerError)
		slog.Error("Failed to write Markdown file", err, slog.String("request-id", reqID))
		return
	}

	slog.Info("New post created successfully", slog.String("request-id", reqID), slog.String("title", post.Title))
	w.WriteHeader(http.StatusCreated)
	changed = true
}

func envVariable(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		slog.Error(err.Error())
		return ""
	}
	slog.Info(os.Getenv(key))
	return os.Getenv(key)
}

var indexText []byte
var changed bool

func indexHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Index hit")
	if changed {
		text, err := parseIndex()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error(err.Error())
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(text)
	} else {
		w.Header().Set("Content-Type", "text/html")
		w.Write(indexText)
	}
}
func parseIndex() ([]byte, error) {
	indexFile, err := os.ReadFile("templates/index.html")
	if err != nil {
		slog.Error(err.Error())
		return []byte(""), err
	}
	tmpl := template.Must(template.New("index").Parse(string(indexFile)))
	posts, err := getPosts()
	if err != nil {
		slog.Error(err.Error())
		return []byte(""), err
	}

	data := struct {
		Latest []Post
		Posts  []Post
	}{
		Latest: posts[:min(len(posts), 1)],
		Posts:  posts,
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		slog.Error(err.Error())
		return []byte(""), err
	}
	return buf.Bytes(), nil
}
func indexInitial() {
	templateText, err := parseIndex()
	if err != nil {
		panic(err)
	}
	indexText = templateText
	changed = false
}

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
func getPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postURL := vars["url"]

	jsonFileName := filepath.Join("posts", postURL+".json")
	jsonData, err := os.ReadFile(jsonFileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		slog.Error(err.Error(), slog.String("url", postURL))
		return
	}

	var post Post
	err = json.Unmarshal(jsonData, &post)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error(err.Error(), slog.String("url", postURL))
		return
	}

	markdownFileName := filepath.Join("posts", postURL+".md")
	markdownData, err := os.ReadFile(markdownFileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error(err.Error(), slog.String("url", postURL))
		return
	}

	htmlContent := mdToHTML(markdownData)

	tem, err := os.ReadFile("templates/post.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error(err.Error(), slog.String("url", postURL))
		return
	}
	tmpl := template.Must(template.New("post").Parse(string(tem)))
	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, struct {
		Title   string
		Date    string
		Content template.HTML
	}{
		Title:   post.Title,
		Date:    post.Date,
		Content: template.HTML(htmlContent),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error(err.Error(), slog.String("url", postURL))
		return
	}
}
func mdToHTML(md []byte) []byte {
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}
func main() {
	indexInitial()
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
	r.HandleFunc("/api/new", newPost).Methods("POST")
	r.HandleFunc("/", indexHandler)
	r.HandleFunc("/posts/{url}", getPost).Methods("GET")
	r.HandleFunc("/index.html", indexHandler)
	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:3000",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	slog.Info("Starting server")
	log.Fatal(srv.ListenAndServe())

}

type Post struct {
	Date    string `json:"date"`
	Title   string `json:"title"`
	Url     string `json:"url"`
	Content string `json:"content"`
}

func getPosts() ([]Post, error) {
	files, err := os.ReadDir("posts")
	if err != nil {
		return nil, err
	}

	var posts []Post
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join("posts", file.Name()))
			if err != nil {
				return nil, err
			}

			var post Post
			err = json.Unmarshal(data, &post)
			if err != nil {
				return nil, err
			}

			posts = append(posts, post)
		}
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date > posts[j].Date
	})

	return posts, nil
}
