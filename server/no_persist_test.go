package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"json-server-go/db"
	"net/http"
	"os"
	"testing"
)

func TestNoPersistPreventsFileWrite(t *testing.T) {
	// Create a temporary DB file
	tmpfile, err := os.CreateTemp("", "db-no-persist-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Initial data
	initial := map[string]interface{}{"posts": []interface{}{map[string]interface{}{"id": "1", "body": "foo"}}}
	dataBytes, _ := json.MarshalIndent(initial, "", "  ")
	if err := ioutil.WriteFile(tmpfile.Name(), dataBytes, 0644); err != nil {
		t.Fatalf("Failed to write temp DB file: %v", err)
	}

	database := db.New(tmpfile.Name())
	if err := database.Load(); err != nil {
		t.Fatalf("Failed to load DB: %v", err)
	}

	config := &Config{IdField: "id", NoPersist: true}
	s := New(database, config)
	s.InitRoutes()

	// POST to create a new post
	newPost := map[string]interface{}{"body": "new post"}
	jsonValue, _ := json.Marshal(newPost)
	req, _ := http.NewRequest("POST", "/posts", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	response := executeRequest(req, s)
	if response.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d", response.Code)
	}

	// Memory should be updated
	reqGet, _ := http.NewRequest("GET", "/posts", nil)
	responseGet := executeRequest(reqGet, s)
	if responseGet.Code != http.StatusOK {
		t.Fatalf("Expected GET /posts to return 200, got %d", responseGet.Code)
	}

	var posts []map[string]interface{}
	json.Unmarshal(responseGet.Body.Bytes(), &posts)
	if len(posts) != 2 {
		t.Fatalf("Expected in-memory DB to have 2 posts, got %d", len(posts))
	}

	// File should be unchanged
	fileContent, err := ioutil.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp DB file: %v", err)
	}
	var fileData map[string]interface{}
	json.Unmarshal(fileContent, &fileData)
	postsFromFile := fileData["posts"].([]interface{})
	if len(postsFromFile) != 1 {
		t.Fatalf("Expected DB file to have 1 post, got %d", len(postsFromFile))
	}
}
