package server

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"json-server-go/db"
)

// Helper to create a fresh DB and Server for each test
func setup() *Server {
	data := map[string]interface{}{
		"posts": []interface{}{
			map[string]interface{}{"id": "1", "body": "foo"},
			map[string]interface{}{"id": "2", "body": "bar"},
		},
		"tags": []interface{}{
			map[string]interface{}{"id": "1", "body": "Technology"},
			map[string]interface{}{"id": "2", "body": "Photography"},
			map[string]interface{}{"id": "3", "body": "photo"},
		},
		"users": []interface{}{
			map[string]interface{}{"id": "1", "username": "Jim", "tel": "0123"},
			map[string]interface{}{"id": "2", "username": "George", "tel": "123"},
		},
		"comments": []interface{}{
			map[string]interface{}{"id": "1", "body": "foo", "published": true, "postId": 1, "userId": 1},
			map[string]interface{}{"id": "2", "body": "bar", "published": false, "postId": 1, "userId": 2},
			map[string]interface{}{"id": "3", "body": "baz", "published": false, "postId": 2, "userId": 1},
			map[string]interface{}{"id": "4", "body": "qux", "published": true, "postId": 2, "userId": 2},
			map[string]interface{}{"id": "5", "body": "quux", "published": false, "postId": 2, "userId": 1},
		},
		"buyers": []interface{}{
			map[string]interface{}{"id": "1", "name": "Aileen", "country": "Colombia", "total": 100},
			map[string]interface{}{"id": "2", "name": "Barney", "country": "Colombia", "total": 200},
			map[string]interface{}{"id": "3", "name": "Carley", "country": "Colombia", "total": 300},
			map[string]interface{}{"id": "4", "name": "Daniel", "country": "Belize", "total": 30},
			map[string]interface{}{"id": "5", "name": "Ellen", "country": "Belize", "total": 20},
			map[string]interface{}{"id": "6", "name": "Frank", "country": "Belize", "total": 10},
			map[string]interface{}{"id": "7", "name": "Grace", "country": "Argentina", "total": 1},
			map[string]interface{}{"id": "8", "name": "Henry", "country": "Argentina", "total": 2},
			map[string]interface{}{"id": "9", "name": "Isabelle", "country": "Argentina", "total": 3},
		},
		"refs": []interface{}{
			map[string]interface{}{"id": "abcd-1234", "url": "http://example.com", "postId": 1, "userId": 1},
		},
		"stringIds": []interface{}{
			map[string]interface{}{"id": "1234"},
		},
		"deep": []interface{}{
			map[string]interface{}{"a": map[string]interface{}{"b": 1}},
			map[string]interface{}{"a": 1},
		},
		"nested": []interface{}{
			map[string]interface{}{"resource": map[string]interface{}{"name": "dewey"}},
			map[string]interface{}{"resource": map[string]interface{}{"name": "cheatem"}},
			map[string]interface{}{"resource": map[string]interface{}{"name": "howe"}},
		},
		"list": []interface{}{
			map[string]interface{}{"id": 1},
			map[string]interface{}{"id": 2},
			map[string]interface{}{"id": 3},
			map[string]interface{}{"id": 4},
			map[string]interface{}{"id": 5},
			map[string]interface{}{"id": 6},
			map[string]interface{}{"id": 7},
			map[string]interface{}{"id": 8},
			map[string]interface{}{"id": 9},
			map[string]interface{}{"id": 10},
			map[string]interface{}{"id": 11},
			map[string]interface{}{"id": 12},
			map[string]interface{}{"id": 13},
			map[string]interface{}{"id": 14},
			map[string]interface{}{"id": 15},
		},
		"profile": map[string]interface{}{
			"name": "typicode",
		},
	}

	database := &db.Database{
		Data: data,
	}
	config := &Config{
		IdField: "id",
	}
	s := New(database, config)
	s.InitRoutes()
	return s
}

func executeRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.ServeHTTP(rr, req)
	return rr
}

func TestQuietFlagDisablesAccessLog(t *testing.T) {
	// Prepare data
	data := map[string]interface{}{
		"posts": []interface{}{
			map[string]interface{}{"id": "1", "body": "foo"},
		},
	}

	database := &db.Database{Data: data}
	config := &Config{IdField: "id", Quiet: true}
	s := New(database, config)
	s.InitRoutes()

	// Capture log output
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	req, _ := http.NewRequest("GET", "/posts", nil)
	executeRequest(req, s)

	if strings.Contains(buf.String(), "GET /posts") {
		t.Errorf("Expected no access log when Quiet is true, but log was present: %s", buf.String())
	}
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func assertJSON(t *testing.T, body []byte, expected interface{}) {
	var actual interface{}
	err := json.Unmarshal(body, &actual)
	if err != nil {
		t.Errorf("Error unmarshalling response body: %v", err)
		return
	}

	// Convert expected to JSON and back to normalize types (e.g. int vs float64)
	expectedBytes, _ := json.Marshal(expected)
	var expectedNormalized interface{}
	json.Unmarshal(expectedBytes, &expectedNormalized)

	if !deepEqual(actual, expectedNormalized) {
		t.Errorf("Expected JSON %v. Got %v", expectedNormalized, actual)
	}
}

// Simple deep equal for JSON objects/arrays
func deepEqual(a, b interface{}) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return string(aBytes) == string(bBytes)
}
