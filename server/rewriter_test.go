package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"json-server-go/db"
)

func setupRewriter(t *testing.T) *Server {
	// Create a temporary routes.json file
	routes := map[string]string{
		"/api/*":                          "/$1",
		"/blog/posts/:id/show":            "/posts/:id",
		"/comments/special/:userId-:body": "/comments/?userId=:userId&body=:body",
		"/firstpostwithcomments":          "/posts/1?_embed=comments",
		"/articles\\?_id=:id":             "/posts/:id",
	}
	file, _ := json.Marshal(routes)
	f, err := os.CreateTemp("", "routes.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Write(file)
	f.Close()

	// Cleanup
	t.Cleanup(func() {
		os.Remove(f.Name())
	})

	data := map[string]interface{}{
		"posts": []interface{}{
			map[string]interface{}{"id": "1", "body": "foo"},
		},
		"comments": []interface{}{
			map[string]interface{}{"id": "1", "body": "foo", "postId": 1, "userId": 1},
			map[string]interface{}{"id": "4", "body": "qux", "postId": 2, "userId": 2},
			map[string]interface{}{"id": "5", "body": "quux", "postId": 2, "userId": 1},
		},
	}

	database := &db.Database{
		Data: data,
	}
	config := &Config{
		Routes:  f.Name(),
		IdField: "id",
	}
	s := New(database, config)
	s.InitRoutes()
	return s
}

func TestRewriter(t *testing.T) {
	s := setupRewriter(t)

	// /api/posts/1 -> /posts/1
	req, _ := http.NewRequest("GET", "/api/posts/1", nil)
	response := executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	var post map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &post)
	if post["id"] != "1" {
		t.Errorf("Expected id 1, got %v", post["id"])
	}

	// /blog/posts/1/show -> /posts/1
	req, _ = http.NewRequest("GET", "/blog/posts/1/show", nil)
	response = executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.Unmarshal(response.Body.Bytes(), &post)
	if post["id"] != "1" {
		t.Errorf("Expected id 1, got %v", post["id"])
	}

	// /comments/special/1-quux -> /comments/?userId=1&body=quux
	// Note: The test data has userId=1, body=quux -> id=5
	req, _ = http.NewRequest("GET", "/comments/special/1-quux", nil)
	response = executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	var comments []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &comments)
	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}
	if fmt.Sprintf("%v", comments[0]["id"]) != "5" {
		t.Errorf("Expected comment id 5, got %v", comments[0]["id"])
	}

	// /firstpostwithcomments -> /posts/1?_embed=comments
	req, _ = http.NewRequest("GET", "/firstpostwithcomments", nil)
	response = executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.Unmarshal(response.Body.Bytes(), &post)
	if post["id"] != "1" {
		t.Errorf("Expected id 1, got %v", post["id"])
	}
	if _, ok := post["comments"]; !ok {
		t.Errorf("Expected comments to be embedded")
	}
}
