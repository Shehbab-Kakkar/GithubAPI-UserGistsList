package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
//	"strconv"
	"strings"
	"time"
)

type GistFile struct {
	Filename string `json:"filename"`
	Language string `json:"language"`
	RawURL   string `json:"raw_url"`
}

type Gist struct {
	ID          string              `json:"id"`
	Description string              `json:"description"`
	HTMLURL     string              `json:"html_url"`
	Files       map[string]GistFile `json:"files"`
}

type Server struct {
	client *http.Client
}

// NewServer returns an HTTP handler
func NewServer() http.Handler {
	s := &Server{
		client: &http.Client{Timeout: 10 * time.Second},
	}
	return http.HandlerFunc(s.handleUserGists)
}

// handleUserGists fetches public gists of a GitHub user with optional pagination
func (s *Server) handleUserGists(w http.ResponseWriter, r *http.Request) {
	user := strings.TrimPrefix(r.URL.Path, "/")
	if user == "" {
		http.Error(w, "user not specified", http.StatusBadRequest)
		return
	}

	// Pagination query params
	page := r.URL.Query().Get("page")
	perPage := r.URL.Query().Get("per_page")

	url := fmt.Sprintf("https://api.github.com/users/%s/gists", user)
	if page != "" || perPage != "" {
		url += "?"
		if page != "" {
			url += "page=" + page
		}
		if perPage != "" {
			if page != "" {
				url += "&"
			}
			url += "per_page=" + perPage
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "golang-gists-api")

	resp, err := s.client.Do(req)
	if err != nil {
		http.Error(w, "failed to contact GitHub", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("GitHub API error: %d", resp.StatusCode), resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read response", http.StatusInternalServerError)
		return
	}

	// Unmarshal GitHub API response
	var rawGists []map[string]interface{}
	if err := json.Unmarshal(body, &rawGists); err != nil {
		http.Error(w, "failed to parse GitHub response", http.StatusInternalServerError)
		return
	}

	// Build a simplified response
	var gists []Gist
	for _, g := range rawGists {
		filesMap := make(map[string]GistFile)
		if files, ok := g["files"].(map[string]interface{}); ok {
			for name, f := range files {
				if fMap, ok := f.(map[string]interface{}); ok {
					filesMap[name] = GistFile{
						Filename: fMap["filename"].(string),
						Language: fmt.Sprintf("%v", fMap["language"]),
						RawURL:   fMap["raw_url"].(string),
					}
				}
			}
		}
		gists = append(gists, Gist{
			ID:          g["id"].(string),
			Description: fmt.Sprintf("%v", g["description"]),
			HTMLURL:     g["html_url"].(string),
			Files:       filesMap,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gists)
}
