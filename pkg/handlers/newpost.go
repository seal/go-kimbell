package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/seal/go-kimbell/pkg/models"
	"github.com/seal/go-kimbell/pkg/utils"
)

func NewPost(w http.ResponseWriter, r *http.Request) {
	uid, err := uuid.NewV6()
	if err != nil {
		http.Error(w, "Failed to generate request ID", http.StatusInternalServerError)
		slog.Error("Failed to generate request ID", err)
		return
	}
	reqID := uid.String()
	slog.Info("New post request received", slog.String("request-id", reqID))
	apiKey := strings.TrimSpace(r.Header.Get("Authorization"))
	if apiKey != utils.EnvVariable("APIKEY") || apiKey == "" {
		http.Error(w, "Unauthorized: Invalid API key", http.StatusUnauthorized)
		slog.Error("Unauthorized request", slog.String("request-id", reqID), slog.String("key", apiKey))
		return
	}
	var post models.Post
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
	err = utils.GeneratePosts()
	if err != nil {
		http.Error(w, "Internal Server Error: Failed to regenerate posts", http.StatusInternalServerError)
		slog.Error("Failed to regenerate posts", err, slog.String("request-id", reqID))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
