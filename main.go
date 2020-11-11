package main

// TODO will be able to put this in a pod with local storage and rsync uploader.

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var serverAddr string
var dataDir string
var sizeLimit int64

var fileDir string
var tempDir string

func init() {
	flag.StringVar(&serverAddr, "addr", ":8000", "address to listen on")
	flag.StringVar(&dataDir, "datadir", "data", "data storage directory")
	flag.Int64Var(&sizeLimit, "sizelimit", 1024*1024*1024, "max file upload size")
}

var fileLocked = make(map[string]bool)
var uploadLock sync.Mutex

func lockFile(name string) bool {
	uploadLock.Lock()
	defer uploadLock.Unlock()

	if len(fileLocked) > 100 || fileLocked[name] {
		return false
	}

	fileLocked[name] = true
	return true
}

func unlockFile(name string) {
	uploadLock.Lock()
	delete(fileLocked, name)
	uploadLock.Unlock()
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	log.Printf("upload %s %s", r.RemoteAddr, r.URL)

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
	if !lockFile(r.URL.Path) {
		http.Error(w, "upload already in progress for file", http.StatusBadRequest)
		return
	}

	defer unlockFile(r.URL.Path)

	if err := os.MkdirAll(tempDir, 0755); err != nil && !os.IsExist(err) {
		log.Printf("failed to create temp directory")
		http.Error(w, "failed to create temp directory", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(fileDir, 0755); err != nil && !os.IsExist(err) {
		log.Printf("failed to create file directory")
		http.Error(w, "failed to create file directory", http.StatusInternalServerError)
		return
	}

	tempPath := filepath.Join(tempDir, r.URL.Path)
	filePath := filepath.Join(fileDir, r.URL.Path)

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
		log.Printf("upload failed %s", tempPath)
		// clean up partial transfer
		os.Remove(tempPath)
		return
	}

	os.Rename(tempPath, filePath)
	log.Printf("upload done %s", filePath)
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, filepath.Join(fileDir, r.URL.Path))
	case "POST":
		uploadFile(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type DataServer struct{}

func (s *DataServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, filepath.Join(fileDir, r.URL.Path))
	case "POST":
		uploadFile(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	flag.Parse()

	fileDir = filepath.Join(dataDir, "file")
	tempDir = filepath.Join(dataDir, "temp")

	server := http.Server{
		Addr:         serverAddr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.Handle("/data/", http.StripPrefix("/data/", &DataServer{}))
	log.Printf("filebin listening on %s", serverAddr)
	log.Fatal(server.ListenAndServe())
}
