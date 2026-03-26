package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port             int           `yaml:"port"`
	FilesDir         string        `yaml:"files_dir"`
	BearerToken      string        `yaml:"bearer_token"`
	MaxIDs           int           `yaml:"max_ids"`
	MaxBodySize      int64         `yaml:"max_body_size"`
	DeleteRetries    int           `yaml:"delete_retries"`
	DeleteRetryDelay time.Duration `yaml:"delete_retry_delay"`
	DAMBaseURL       string        `yaml:"dam_base_url"`
	DAMBearerToken   string        `yaml:"dam_bearer_token"`
	DAMTimeout       time.Duration `yaml:"dam_timeout"`
}

var cfg Config
var validID = regexp.MustCompile(`^\d{6}$|^\d{9}$`)
var fileIDPattern = regexp.MustCompile(`(?:preview)?(\d{6}|\d{9})\.jpg$`)

func loadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open config file: %w", err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true) // error on unknown fields
	if err := decoder.Decode(&cfg); err != nil {
		return fmt.Errorf("could not parse config file: %w", err)
	}

	// Sanity checks
	if cfg.Port == 0 {
		return fmt.Errorf("port must be set")
	}
	if cfg.FilesDir == "" {
		return fmt.Errorf("files_dir must be set")
	}
	if cfg.BearerToken == "" {
		return fmt.Errorf("bearer_token must be set")
	}
	if cfg.MaxIDs <= 0 {
		return fmt.Errorf("max_ids must be greater than 0")
	}
	if cfg.MaxBodySize <= 0 {
		return fmt.Errorf("max_body_size must be greater than 0")
	}
	if cfg.DeleteRetries <= 0 {
		return fmt.Errorf("delete_retries must be greater than 0")
	}
	if cfg.DeleteRetryDelay <= 0 {
		return fmt.Errorf("delete_retry_delay must be greater than 0")
	}
	if cfg.DAMBaseURL == "" {
		return fmt.Errorf("dam_base_url must be set")
	}
	if cfg.DAMBearerToken == "" {
		return fmt.Errorf("dam_bearer_token must be set")
	}
	if cfg.DAMTimeout <= 0 {
		return fmt.Errorf("dam_timeout must be greater than 0")
	}

	return nil
}

type deleteRequest struct {
	IDs []string `json:"ids"`
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != cfg.BearerToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request body size
	r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBodySize)

	var req deleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid or oversized JSON body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if len(req.IDs) > cfg.MaxIDs {
		http.Error(w, fmt.Sprintf("too many IDs: max %d allowed", cfg.MaxIDs), http.StatusBadRequest)
		return
	}

	// Validate all IDs
	for _, id := range req.IDs {
		if id == "" {
			http.Error(w, "ID must not be empty", http.StatusBadRequest)
			return
		}
		if !validID.MatchString(id) {
			http.Error(w, "invalid product ID: "+id, http.StatusBadRequest)
			return
		}

		// Guard against path traversal even though regex already prevents it

		if filepath.Base(id) != id {
			http.Error(w, "invalid product ID: "+id, http.StatusBadRequest)
			return
		}
	}

	// Respond 200 immediately, delete in background
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	go func(ids []string) {
		for _, id := range ids {
			deleteFiles(id)
		}
	}(req.IDs)
}

func deleteFiles(id string) {
	targets := []string{
		filepath.Join(cfg.FilesDir, id+".jpg"),
		filepath.Join(cfg.FilesDir, "preview"+id+".jpg"),
	}

	for _, path := range targets {
		if err := removeWithRetry(path); err != nil {
			log.Printf("failed to delete %s after %d attempts: %v", path, cfg.DeleteRetries, err)
		}
	}
}

func removeWithRetry(path string) error {
	for attempt := range cfg.DeleteRetries {
		err := os.Remove(path)
		if err == nil {
			log.Printf("deleted %s", path)
			return nil
		}
		if os.IsNotExist(err) {
			return nil // file doesn't exist, nothing to do
		}
		log.Printf("delete attempt %d failed for %s: %v", attempt+1, path, err)
		time.Sleep(cfg.DeleteRetryDelay)
	}
	return fmt.Errorf("file still locked after %d attempts", cfg.DeleteRetries)
}

// --- /api/refresh ---

func refreshHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	go runRefresh()
}

func runRefresh() {
	entries, err := os.ReadDir(cfg.FilesDir)
	if err != nil {
		log.Printf("refresh: failed to read files_dir: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -2)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		info, err := entry.Info()
		if err != nil {
			log.Printf("refresh: could not stat %s: %v", name, err)
			continue
		}

		if info.ModTime().After(cutoff) {
			continue
		}

		path := filepath.Join(cfg.FilesDir, name)

		// Delete the file
		if err := removeWithRetry(path); err != nil {
			log.Printf("refresh: failed to delete %s: %v", path, err)
			continue
		}

		matches := fileIDPattern.FindStringSubmatch(name)
		if matches != nil {
			callDAMAPI(matches[1])
		}
	}
}

func callDAMAPI(id string) {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(cfg.DAMBaseURL, "/"), id)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DAMTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("dam api: failed to create request for ID %s: %v", id, err)

		return
	}
	req.Header.Set("Authorization", "Bearer "+cfg.DAMBearerToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("dam api: request failed for ID %s: %v", id, err)
		return
	}
	defer resp.Body.Close()

	log.Printf("dam api: called for ID %s, status %d", id, resp.StatusCode)
}

func main() {
	if err := loadConfig("config.yaml"); err != nil {
		log.Fatalf("config error: %v", err)
	}

	http.HandleFunc("/api/delete", authMiddleware(deleteHandler))
	http.HandleFunc("/api/refresh", authMiddleware(refreshHandler))

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
