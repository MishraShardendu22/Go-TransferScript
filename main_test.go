package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"
)

func TestTransferRepo(t *testing.T) {
	// Helper function to create a new resty client for testing.
	// Note: SetHostURL is not strictly necessary here anymore since transferRepo takes a full URL,
	// but it's good practice if other client methods were to be used with relative paths.
	newTestClient := func() *resty.Client {
		client := resty.New()
		return client
	}

	// Define test cases
	tests := []struct {
		name            string
		serverHandler   http.HandlerFunc // Simulates GitHub API server responses
		newUser         string
		repoName        string // Name of the repo for transfer body and logging
		githubToken     string
		expectError     bool
		expectedErrorMsgSubstring string // If expectError is true, check if the error message contains this
		// originalUser and repo for URL construction are now part of how transferURL is built per test case
		urlOriginalUser string // For constructing the test URL path
		urlRepo         string // For constructing the test URL path
	}{
		{
			name: "Successful transfer",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
					t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", auth)
				}
				// Path check: /repos/test-orig-user/test-repo/transfer
				expectedPath := "/repos/test-orig-user/test-repo/transfer"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected URL path '%s', got '%s'", expectedPath, r.URL.Path)
				}
				w.WriteHeader(http.StatusAccepted) // 202
				fmt.Fprintln(w, `{"message": "Repository transfer initiated"}`)
			},
			newUser:         "test-new-user",
			repoName:        "test-repo",
			githubToken:     "test-token",
			expectError:     false,
			urlOriginalUser: "test-orig-user",
			urlRepo:         "test-repo",
		},
		{
			name: "Unauthorized - 401",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized) // 401
				fmt.Fprintln(w, `{"message": "Bad credentials"}`)
			},
			newUser:         "test-new-user",
			repoName:        "test-repo-401",
			githubToken:     "invalid-token",
			expectError:     true,
			expectedErrorMsgSubstring: "Unauthorized (HTTP 401)",
			urlOriginalUser: "test-orig-user",
			urlRepo:         "test-repo-401",
		},
		{
			name: "Forbidden - 403",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden) // 403
				fmt.Fprintln(w, `{"message": "Rate limit exceeded or insufficient permissions"}`)
			},
			newUser:         "test-new-user",
			repoName:        "test-repo-403",
			githubToken:     "test-token",
			expectError:     true,
			expectedErrorMsgSubstring: "Forbidden (HTTP 403)",
			urlOriginalUser: "test-orig-user",
			urlRepo:         "test-repo-403",
		},
		{
			name: "Not Found - 404",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound) // 404
				fmt.Fprintln(w, `{"message": "Repository not found"}`)
			},
			newUser:         "test-new-user",
			repoName:        "non-existent-repo",
			githubToken:     "test-token",
			expectError:     true,
			expectedErrorMsgSubstring: "Repository or user not found (HTTP 404)",
			urlOriginalUser: "test-orig-user",
			urlRepo:         "non-existent-repo",
		},
		{
			name: "Unprocessable Entity - 422",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity) // 422
				fmt.Fprintln(w, `{"message": "Validation failed"}`)
			},
			newUser:         "test-new-user",
			repoName:        "test-repo-422",
			githubToken:     "test-token",
			expectError:     true,
			expectedErrorMsgSubstring: "Unprocessable Entity (HTTP 422)",
			urlOriginalUser: "test-orig-user",
			urlRepo:         "test-repo-422",
		},
		{
			name: "Unexpected Status Code - 500",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError) // 500
				fmt.Fprintln(w, `{"message": "Server error"}`)
			},
			newUser:         "test-new-user",
			repoName:        "test-repo-500",
			githubToken:     "test-token",
			expectError:     true,
			expectedErrorMsgSubstring: "Unexpected status code (HTTP 500)",
			urlOriginalUser: "test-orig-user",
			urlRepo:         "test-repo-500",
		},
		{
			name:            "Client request error (e.g. network error)",
			serverHandler:   nil, // No server needed, client will fail using a bad URL
			newUser:         "test-new-user",
			repoName:        "test-repo-network-error",
			githubToken:     "test-token",
			expectError:     true,
			expectedErrorMsgSubstring: "failed to send transfer request",
			urlOriginalUser: "test-orig-user", // Used to construct the bad URL
			urlRepo:         "test-repo-network-error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient() // Create a new client for each test
			var transferURL string

			if tc.serverHandler != nil {
				server := httptest.NewServer(tc.serverHandler)
				defer server.Close()
				// Construct the URL to use the test server, mimicking the GitHub API path structure
				transferURL = fmt.Sprintf("%s/repos/%s/%s/transfer", server.URL, tc.urlOriginalUser, tc.urlRepo)
			} else {
				// For client-side error test, use an invalid URL that won't resolve or connect
				transferURL = fmt.Sprintf("http://.invalidlocaldomain:12345/repos/%s/%s/transfer", tc.urlOriginalUser, tc.urlRepo)
			}

			err := transferRepo(client, transferURL, tc.newUser, tc.repoName, tc.githubToken)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				} else if !strings.Contains(err.Error(), tc.expectedErrorMsgSubstring) {
					t.Errorf("Expected error message to contain '%s', but got '%s'", tc.expectedErrorMsgSubstring, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}
		})
	}
}
