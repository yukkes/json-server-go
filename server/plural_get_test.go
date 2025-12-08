package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestGetDb(t *testing.T) {
	s := setup()
	req, _ := http.NewRequest("GET", "/db", nil)
	response := executeRequest(req, s)

	checkResponseCode(t, http.StatusOK, response.Code)
	// Check content type?
	// assertJSON(t, response.Body.Bytes(), s.DB.GetData()) // This might be too big/complex to match exactly if map order varies, but JSON marshal should handle it.
}

func TestGetPlural(t *testing.T) {
	s := setup()
	req, _ := http.NewRequest("GET", "/posts", nil)
	response := executeRequest(req, s)

	checkResponseCode(t, http.StatusOK, response.Code)

	expected := []map[string]interface{}{
		{"id": "1", "body": "foo"},
		{"id": "2", "body": "bar"},
	}
	assertJSON(t, response.Body.Bytes(), expected)
}

func TestGetPluralNotFound(t *testing.T) {
	s := setup()
	req, _ := http.NewRequest("GET", "/undefined", nil)
	response := executeRequest(req, s)

	checkResponseCode(t, http.StatusNotFound, response.Code)
}

func TestGetPluralFilter(t *testing.T) {
	s := setup()

	// Filter by property
	req, _ := http.NewRequest("GET", "/comments?postId=1&published=true", nil)
	response := executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)

	expected := []map[string]interface{}{
		{"id": "1", "body": "foo", "published": true, "postId": 1, "userId": 1},
	}
	assertJSON(t, response.Body.Bytes(), expected)

	// Strict filter
	req, _ = http.NewRequest("GET", "/users?tel=123", nil)
	response = executeRequest(req, s)
	expectedUsers := []map[string]interface{}{
		{"id": "2", "username": "George", "tel": "123"},
	}
	assertJSON(t, response.Body.Bytes(), expectedUsers)

	// Multiple filters (same key) - Not supported by standard net/url Query() easily for map[string]string,
	// but json-server supports array. Go's r.URL.Query() returns map[string][]string.
	// Let's see if our implementation supports it.
	req, _ = http.NewRequest("GET", "/comments?id=1&id=2", nil)
	response = executeRequest(req, s)
	expectedComments := []map[string]interface{}{
		{"id": "1", "body": "foo", "published": true, "postId": 1, "userId": 1},
		{"id": "2", "body": "bar", "published": false, "postId": 1, "userId": 2},
	}
	assertJSON(t, response.Body.Bytes(), expectedComments)

	// Deep filter
	req, _ = http.NewRequest("GET", "/deep?a.b=1", nil)
	response = executeRequest(req, s)
	expectedDeep := []map[string]interface{}{
		{"a": map[string]interface{}{"b": 1}},
	}
	assertJSON(t, response.Body.Bytes(), expectedDeep)
}

func TestGetPluralSearch(t *testing.T) {
	s := setup()

	// Full-text search
	req, _ := http.NewRequest("GET", "/tags?q=pho", nil)
	response := executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)

	expected := []map[string]interface{}{
		{"id": "2", "body": "Photography"},
		{"id": "3", "body": "photo"},
	}
	assertJSON(t, response.Body.Bytes(), expected)

	// Search with other params
	req, _ = http.NewRequest("GET", "/comments?q=qu&published=true", nil)
	response = executeRequest(req, s)
	expectedComments := []map[string]interface{}{
		{"id": "4", "body": "qux", "published": true, "postId": 2, "userId": 2},
	}
	assertJSON(t, response.Body.Bytes(), expectedComments)
}

func TestGetPluralSort(t *testing.T) {
	s := setup()

	// Sort by field
	req, _ := http.NewRequest("GET", "/tags?_sort=body", nil)
	response := executeRequest(req, s)

	expected := []map[string]interface{}{
		{"id": "2", "body": "Photography"},
		{"id": "1", "body": "Technology"},
		{"id": "3", "body": "photo"},
	}
	assertJSON(t, response.Body.Bytes(), expected)

	// Sort desc
	req, _ = http.NewRequest("GET", "/tags?_sort=body&_order=DESC", nil)
	response = executeRequest(req, s)
	expectedDesc := []map[string]interface{}{
		{"id": "3", "body": "photo"},
		{"id": "1", "body": "Technology"},
		{"id": "2", "body": "Photography"},
	}
	assertJSON(t, response.Body.Bytes(), expectedDesc)

	// Sort numerical
	req, _ = http.NewRequest("GET", "/posts?_sort=id&_order=DESC", nil)
	response = executeRequest(req, s)
	expectedPosts := []map[string]interface{}{
		{"id": "2", "body": "bar"},
		{"id": "1", "body": "foo"},
	}
	assertJSON(t, response.Body.Bytes(), expectedPosts)

	// Sort multiple fields
	req, _ = http.NewRequest("GET", "/buyers?_sort=country,total&_order=asc,desc", nil)
	response = executeRequest(req, s)

	var buyers []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &buyers)

	expectedIds := []string{"9", "8", "7", "4", "5", "6", "3", "2", "1"}
	if len(buyers) != 9 {
		t.Errorf("Expected 9 buyers, got %d", len(buyers))
	} else {
		for i, id := range expectedIds {
			if fmt.Sprintf("%v", buyers[i]["id"]) != id {
				t.Errorf("Index %d: Expected id %s, got %v", i, id, buyers[i]["id"])
			}
		}
	}
}

func TestGetPluralSlice(t *testing.T) {
	s := setup()

	// _end
	req, _ := http.NewRequest("GET", "/comments?_end=2", nil)
	response := executeRequest(req, s)

	if response.Header().Get("X-Total-Count") != "5" {
		t.Errorf("Expected X-Total-Count 5, got %s", response.Header().Get("X-Total-Count"))
	}

	var comments []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &comments)
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}

	// _start & _end
	req, _ = http.NewRequest("GET", "/comments?_start=1&_end=2", nil)
	response = executeRequest(req, s)
	json.Unmarshal(response.Body.Bytes(), &comments)
	if len(comments) != 1 { // index 1 to 2 (exclusive) -> 1 item
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}
	if fmt.Sprintf("%v", comments[0]["id"]) != "2" {
		t.Errorf("Expected comment id 2, got %v", comments[0]["id"])
	}

	// _start & _limit
	req, _ = http.NewRequest("GET", "/comments?_start=1&_limit=1", nil)
	response = executeRequest(req, s)
	json.Unmarshal(response.Body.Bytes(), &comments)
	if len(comments) != 1 {
		t.Errorf("Expected 1 comment, got %d", len(comments))
	}
	if fmt.Sprintf("%v", comments[0]["id"]) != "2" {
		t.Errorf("Expected comment id 2, got %v", comments[0]["id"])
	}
}

func TestGetPluralPagination(t *testing.T) {
	s := setup()

	// _page=2 (default limit 10)
	// list has 15 items. Page 1: 1-10, Page 2: 11-15
	req, _ := http.NewRequest("GET", "/list?_page=2", nil)
	response := executeRequest(req, s)

	var list []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &list)
	if len(list) != 5 {
		t.Errorf("Expected 5 items, got %d", len(list))
	}

	// Link header
	// _page=2&_limit=1
	// list has 15 items.
	// first: 1, prev: 1, next: 3, last: 15
	req, _ = http.NewRequest("GET", "/list?_page=2&_limit=1", nil)
	response = executeRequest(req, s)
	link := response.Header().Get("Link")
	if !strings.Contains(link, `rel="first"`) || !strings.Contains(link, `rel="prev"`) || !strings.Contains(link, `rel="next"`) || !strings.Contains(link, `rel="last"`) {
		t.Errorf("Link header missing relations: %s", link)
	}
}

func TestGetPluralOperators(t *testing.T) {
	s := setup()

	// _gte, _lte
	req, _ := http.NewRequest("GET", "/comments?id_gte=2&id_lte=3", nil)
	response := executeRequest(req, s)
	var comments []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &comments)
	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}
	// Should be id 2 and 3

	// _ne
	req, _ = http.NewRequest("GET", "/comments?id_ne=1", nil)
	response = executeRequest(req, s)
	json.Unmarshal(response.Body.Bytes(), &comments)
	if len(comments) != 4 {
		t.Errorf("Expected 4 comments, got %d", len(comments))
	}

	// _like
	req, _ = http.NewRequest("GET", "/tags?body_like=photo", nil)
	response = executeRequest(req, s)
	var tags []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &tags)
	if len(tags) != 2 { // Photography, photo
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}
}

func TestGetPluralById(t *testing.T) {
	s := setup()
	req, _ := http.NewRequest("GET", "/posts/1", nil)
	response := executeRequest(req, s)

	checkResponseCode(t, http.StatusOK, response.Code)

	expected := map[string]interface{}{"id": "1", "body": "foo"}
	assertJSON(t, response.Body.Bytes(), expected)
}

func TestGetPluralRelationships(t *testing.T) {
	s := setup()

	// _embed
	req, _ := http.NewRequest("GET", "/posts?_embed=comments", nil)
	response := executeRequest(req, s)

	var posts []map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &posts)

	if len(posts) > 0 {
		if _, ok := posts[0]["comments"]; !ok {
			t.Errorf("Expected comments to be embedded")
		}
	}

	// _expand
	req, _ = http.NewRequest("GET", "/comments/1?_expand=post", nil)
	response = executeRequest(req, s)

	var comment map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &comment)

	if _, ok := comment["post"]; !ok {
		t.Errorf("Expected post to be expanded")
	}
}
