package handlers

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

func GetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postURL := vars["url"]
	htmlFileName := filepath.Join("generated", postURL+".html")
	htmlContent, err := os.ReadFile(htmlFileName)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Post not found", http.StatusNotFound)
			slog.Error("Post not found", slog.String("url", postURL))
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			slog.Error("Failed to read generated post", err, slog.String("url", postURL))
		}
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(htmlContent)
}
