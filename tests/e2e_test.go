package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

const baseURL = "http://localhost:8080"


type Team struct {
	Name string `json:"team_name"`
}


type User struct {
	ID string `json:"user_id"`
}


type PR struct {
	ID        string   `json:"pull_request_id"`
	Status    string   `json:"status"`
	Reviewers []string `json:"assigned_reviewers"`
}


func TestHappyPath(t *testing.T) {
	time.Sleep(1 * time.Second)

	client := &http.Client{Timeout: 5 * time.Second}

	t.Log("Step 1: Creating Team")

	teamPayload := `{"team_name": "gophers", "members": [
		{"user_id": "u1", "username": "Alice", "is_active": true},
		{"user_id": "u2", "username": "Bob", "is_active": true},
		{"user_id": "u3", "username": "Charlie", "is_active": true}
	]}`

	resp, err := client.Post(baseURL+"/team/add", "application/json", bytes.NewBuffer([]byte(teamPayload)))

	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusBadRequest { 
		t.Fatalf("Expected 201 or 400, got %d", resp.StatusCode)
	}

	t.Log("Step 2: Creating PR")
	prID := fmt.Sprintf("pr-%d", time.Now().Unix())
	prPayload := fmt.Sprintf(`{"pull_request_id": "%s", "pull_request_name": "Fix", "author_id": "u1"}`, prID)
	
	resp, err = client.Post(baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer([]byte(prPayload)))
	if err != nil { 
		t.Fatal(err) 
	}
	
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to create PR: status %d", resp.StatusCode)
	}

	var prResp struct {
		PR PR `json:"pr"`
	}
	json.NewDecoder(resp.Body).Decode(&prResp)

	if len(prResp.PR.Reviewers) != 2 {
		t.Errorf("Expected 2 reviewers, got %d", len(prResp.PR.Reviewers))
	}
	t.Logf("Assigned reviewers: %v", prResp.PR.Reviewers)

	t.Log("Step 3: Merging PR")
	mergePayload := fmt.Sprintf(`{"pull_request_id": "%s"}`, prID)
	resp, _ = client.Post(baseURL+"/pullRequest/merge", "application/json", bytes.NewBuffer([]byte(mergePayload)))
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Merge failed: %d", resp.StatusCode)
	}
}