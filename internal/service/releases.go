package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ReleaseCategory is a group of releases (e.g. "OLake", "OLake Helm").
type ReleaseCategory struct {
	Name     string          `json:"name"`
	Releases []ReleaseEntry  `json:"releases"`
}

// ReleaseEntry is a single version in a category.
type ReleaseEntry struct {
	Version     string `json:"version"`
	Date        string `json:"date"`
	Description string `json:"description,omitempty"`
	Link        string `json:"link,omitempty"`
}

// FetchReleases downloads and parses a releases.json from the given URL.
// The expected format is an array of ReleaseCategory objects:
//
//	[
//	  {
//	    "name": "OLake",
//	    "releases": [
//	      {"version": "v1.2.0", "date": "2026-03-01", "description": "..."}
//	    ]
//	  }
//	]
//
// Returns nil (no error) if url is empty — air-gapped mode.
// Returns nil with a 5-second timeout on network errors (non-fatal).
func FetchReleases(url string) ([]ReleaseCategory, error) {
	if url == "" {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil // network error — silently skip in air-gapped
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // non-200 — silently skip
	}

	var categories []ReleaseCategory
	if err := json.NewDecoder(resp.Body).Decode(&categories); err != nil {
		return nil, fmt.Errorf("parse releases.json: %w", err)
	}

	return categories, nil
}
