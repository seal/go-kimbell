package utils

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/joho/godotenv"
	"github.com/seal/go-kimbell/pkg/models"
)

func EnvVariable(key string) string {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Failed to open log file", err)
		os.Exit(1)
	}
	return os.Getenv(key)
}
func ParseIndex() error {
	indexFile, err := os.ReadFile("templates/index.html")
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	tmpl := template.Must(template.New("index").Parse(string(indexFile)))
	posts, _, err := GetPosts()
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	data := struct {
		Latest []models.Post
		Posts  []models.Post
	}{
		Latest: posts[:min(len(posts), 1)],
		Posts:  posts,
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	err = os.MkdirAll("static", os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile("static/index.html", buf.Bytes(), 0644)
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	return nil
}
func GeneratePosts() error {
	posts, _, err := GetPosts()
	if err != nil {
		return err
	}

	err = os.MkdirAll("static", os.ModePerm)
	if err != nil {
		return err
	}

	for _, post := range posts {
		markdownFileName := filepath.Join("posts", post.Url+".md")
		markdownData, err := os.ReadFile(markdownFileName)
		if err != nil {
			return err
		}

		htmlContent := mdToHTML(markdownData)

		tem, err := os.ReadFile("templates/post.html")
		if err != nil {
			return err
		}
		tmpl := template.Must(template.New("post").Parse(string(tem)))
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, struct {
			Title   string
			Date    string
			Content template.HTML
		}{
			Title:   post.Title,
			Date:    post.Date,
			Content: template.HTML(htmlContent),
		})
		if err != nil {
			return err
		}

		htmlFileName := filepath.Join("static", post.Url+".html")
		err = os.WriteFile(htmlFileName, buf.Bytes(), 0644)
		if err != nil {
			return err
		}
	}

	return nil
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
func GetPosts() ([]models.Post, time.Time, error) {
	files, err := os.ReadDir("posts")
	if err != nil {
		return nil, time.Time{}, err
	}

	var posts []models.Post
	var latestModTime time.Time

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join("posts", file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, time.Time{}, err
			}

			var post models.Post
			err = json.Unmarshal(data, &post)
			if err != nil {
				return nil, time.Time{}, err
			}

			posts = append(posts, post)

			fileInfo, err := os.Stat(filePath)
			if err != nil {
				return nil, time.Time{}, err
			}
			if fileInfo.ModTime().After(latestModTime) {
				latestModTime = fileInfo.ModTime()
			}
		}
	}

	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date > posts[j].Date
	})

	return posts, latestModTime, nil
}
