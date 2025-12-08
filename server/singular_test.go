package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGetSingular(t *testing.T) {
	s := setup()
	req, _ := http.NewRequest("GET", "/profile", nil)
	response := executeRequest(req, s)

	checkResponseCode(t, http.StatusOK, response.Code)

	var profile map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &profile)

	if profile["name"] != "typicode" {
		t.Errorf("Expected name 'typicode'. Got %v", profile["name"])
	}
}

func TestUpdateSingular(t *testing.T) {
	s := setup()
	update := map[string]interface{}{"name": "updated"}
	jsonValue, _ := json.Marshal(update)

	// POST
	req, _ := http.NewRequest("POST", "/profile", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	response := executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	var profile map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &profile)
	if profile["name"] != "updated" {
		t.Errorf("Expected name 'updated'. Got %v", profile["name"])
	}

	// PUT
	update["name"] = "updated put"
	jsonValue, _ = json.Marshal(update)
	req, _ = http.NewRequest("PUT", "/profile", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	response = executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.Unmarshal(response.Body.Bytes(), &profile)
	if profile["name"] != "updated put" {
		t.Errorf("Expected name 'updated put'. Got %v", profile["name"])
	}

	// PATCH
	update["name"] = "updated patch"
	jsonValue, _ = json.Marshal(update)
	req, _ = http.NewRequest("PATCH", "/profile", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	response = executeRequest(req, s)
	checkResponseCode(t, http.StatusOK, response.Code)
	json.Unmarshal(response.Body.Bytes(), &profile)
	if profile["name"] != "updated patch" {
		t.Errorf("Expected name 'updated patch'. Got %v", profile["name"])
	}
}
