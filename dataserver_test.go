package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const testRootDir = "dataserver_test"

func mustCleanDir(dir string) {
	err := os.RemoveAll(dir)
	if err != nil {
		panic(err)
	}
}

func mustReadAll(r io.Reader) []byte {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return b
}

func downloadFile(handler http.Handler, url string) *http.Response {
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
}

func uploadFile(handler http.Handler, url string, body []byte) *http.Response {
	req := httptest.NewRequest("POST", url, bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
}

func testUploadDownload(t *testing.T, handler http.Handler, url string, data []byte) {
	if resp := downloadFile(handler, url); resp.StatusCode != http.StatusNotFound {
		t.Errorf("file should not exist yet")
	}

	if resp := uploadFile(handler, url, data); resp.StatusCode != http.StatusOK {
		t.Errorf("file upload failed")
	}

	if resp := uploadFile(handler, url, data); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("file upload should have failed")
	}

	if resp := downloadFile(handler, url); resp.StatusCode != http.StatusOK {
		t.Errorf("expecting http status OK")
	} else {
		b := mustReadAll(resp.Body)
		if bytes.Compare(data, b) != 0 {
			t.Errorf("download differs from upload")
		}
	}
}

func TestUploadDownload(t *testing.T) {
	mustCleanDir(testRootDir)
	defer mustCleanDir(testRootDir)

	handler := http.StripPrefix("/data/", DataServer(testRootDir))

	for i := 0; i < 100; i++ {
		url := fmt.Sprintf("http://example.com/data/file-%d", i)
		data := []byte(fmt.Sprintf("here's some test data - test %d", i))
		testUploadDownload(t, handler, url, data)
	}

	url := "http://example.com/data/largefile"
	data := make([]byte, 256*1024*1024)
	rand.Read(data)
	testUploadDownload(t, handler, url, data)
}
