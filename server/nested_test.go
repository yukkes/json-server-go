package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestNested(t *testing.T) {
	s := setup()

	// GET /posts/1/comments
	req, _ := http.NewRequest("GET", "/posts/1/comments", nil)
	response := executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)

	var comments []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &comments)

	if len(comments) != 2 {
		t.Errorf("Expected 2 comments. Got %d", len(comments))
	}
	// Should be comments with postId=1 (id 1 and 2)
	for _, c := range comments {
		if c["postId"] != 1.0 && c["postId"] != "1" { // JSON number is float64
			t.Errorf("Expected postId 1. Got %v", c["postId"])
		}
	}
}
