package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/seal/go-kimbell/pkg/utils"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("Index hit")

	_, latestModTime, err := utils.GetPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		slog.Error(err.Error())
		return
	}

	if latestModTime.After(lastGenerationTime) {
		err := utils.ParseIndex()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error(err.Error())
			return
		}
		lastGenerationTime = latestModTime
	}
	http.ServeFile(w, r, "static/index.html")
}

var lastGenerationTime time.Time
