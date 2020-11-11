package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const sizeLimit = 1024 * 1024 * 1024

// DataServer provides an upload / download API and manages all of its
// data inside the root directory.
func DataServer(root string) http.Handler {
	server := &dataServer{
		root:    root,
		fileDir: filepath.Join(root, "file"),
		tempDir: filepath.Join(root, "temp"),
	}
	server.busy.M = make(map[string]bool)
	return server
}

type dataServer struct {
	root    string
	fileDir string
	tempDir string
	busy    struct {
		M map[string]bool
		sync.Mutex
	}
}

func (s *dataServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		http.ServeFile(w, r, filepath.Join(s.fileDir, r.URL.Path))
	case http.MethodPost:
		s.uploadFile(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *dataServer) lockFile(name string) bool {
	s.busy.Lock()
	defer s.busy.Unlock()
	if len(s.busy.M) > 100 || s.busy.M[name] {
		return false
	}
	s.busy.M[name] = true
	return true
}

func (s *dataServer) unlockFile(name string) {
	s.busy.Lock()
	delete(s.busy.M, name)
	s.busy.Unlock()
}

func (s *dataServer) uploadFile(w http.ResponseWriter, r *http.Request) {
	// uploads must include size
	if r.ContentLength < 0 {
		http.Error(w, "content-length is required", http.StatusBadRequest)
		return
	}

	// uploads cannot exceed size limit
	if r.ContentLength > sizeLimit {
		http.Error(w, fmt.Sprintf("content-length must be less than %d", sizeLimit), http.StatusBadRequest)
		return
	}

	// uploads cannot contain slashes
	if strings.Count(r.URL.Path, "/") > 1 {
		http.Error(w, "filename must not contain slashes", http.StatusBadRequest)
		return
	}

	// upload must not be in progress for this file
	if !s.lockFile(r.URL.Path) {
		http.Error(w, "upload already in progress for file", http.StatusBadRequest)
		return
	}

	defer s.unlockFile(r.URL.Path)

	log.Printf("uploading \"%s\" (%d bytes) from %s", r.URL.Path, r.ContentLength, r.RemoteAddr)

	if err := os.MkdirAll(s.tempDir, 0755); err != nil && !os.IsExist(err) {
		log.Printf("failed to create temp directory")
		http.Error(w, "failed to create temp directory", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(s.fileDir, 0755); err != nil && !os.IsExist(err) {
		log.Printf("failed to create file directory")
		http.Error(w, "failed to create file directory", http.StatusInternalServerError)
		return
	}

	tempPath := filepath.Join(s.tempDir, r.URL.Path)
	filePath := filepath.Join(s.fileDir, r.URL.Path)

	if _, err := os.Stat(filePath); err == nil {
		http.Error(w, "file already exists", http.StatusBadRequest)
		return
	}

	f, err := os.OpenFile(tempPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("could not open temp file %s", tempPath)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.CopyN(f, r.Body, r.ContentLength); err != nil {
		log.Printf("upload error \"%s\": %s", r.URL.Path, err)
		// clean up partial transfer
		os.Remove(tempPath)
		return
	}

	// atomic move temp file to final destination
	if err := os.Rename(tempPath, filePath); err != nil {
		log.Printf("upload error \"%s\": %s", r.URL.Path, err)
		return
	}

	log.Printf("upload ok \"%s\"", r.URL.Path)
}
