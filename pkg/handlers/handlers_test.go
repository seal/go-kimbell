package handlers_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/seal/go-kimbell/pkg/handlers"
)

func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		panic("Failed to load .env file")
	}
	err = os.Chdir("../..")
	if err != nil {
		panic(err)
	}

}

func TestNewPost(t *testing.T) {
	payload := []byte(`{
		"date": "2024-01-08",
		"title": "Test Post",
		"url": "test-post",
		"content": "# Test Post\n\nThis is a test post."
	}`)
	req, err := http.NewRequest("POST", "/api/new", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", os.Getenv("APIKEY")) // Valid api_key

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.NewPost)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	req.Header.Set("Authorization", "invalid-api-key")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
	}

	// Missing fields
	payload = []byte(`{
		"date": "2024-01-08",
		"title": "Test Post"
	}`)
	req, err = http.NewRequest("POST", "/api/new", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", os.Getenv("APIKEY"))

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestGetPost(t *testing.T) {
	req, err := http.NewRequest("GET", "/posts/test-post", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/posts/{url}", handlers.GetPost)
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	req, err = http.NewRequest("GET", "/posts/non-existing-post", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	defer func() {
		// Delete test post files
		os.Remove("posts/test-post.json")
		os.Remove("posts/test-post.md")
		os.Remove("generated/test-post.html")
	}()
}
func TestIndexPageChange(t *testing.T) {
	initialIndexContent, err := os.ReadFile("generated/index.html")
	if err != nil {
		t.Fatalf("Failed to read initial index page: %v", err)
	}

	payload := []byte(`{
		"date": "2024-01-08",
		"title": "Test Post",
		"url": "test-post",
		"content": "# Test Post\n\nThis is a test post."
	}`)
	req, err := http.NewRequest("POST", "/api/new", bytes.NewBuffer(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", os.Getenv("APIKEY"))

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handlers.NewPost)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	// Hit / ep to trigger index change
	req, err = http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(handlers.IndexHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	updatedIndexContent, err := os.ReadFile("generated/index.html")
	if err != nil {
		t.Fatalf("Failed to read updated index page: %v", err)
	}

	if bytes.Equal(initialIndexContent, updatedIndexContent) {
		t.Error("Index page content did not change after generating a new post")
	}

	defer func() {
		// Delete test post files
		os.Remove("posts/test-post.json")
		os.Remove("posts/test-post.md")
		os.Remove("generated/test-post.html")
	}()
}
