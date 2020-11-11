package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const testRootDir = "dataserver_test"

func downloadFile(handler http.Handler, url string) ([]byte, int, error) {
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	b, err := ioutil.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func uploadFile(handler http.Handler, url string, body []byte) ([]byte, int, error) {
	req := httptest.NewRequest("POST", url, bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	resp := w.Result()
	b, err := ioutil.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func testUploadDownload(t *testing.T, handler http.Handler, url string, data []byte) {
	_, code, err := downloadFile(handler, url)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusNotFound {
		t.Errorf("file should not exist yet")
	}

	_, code, err = uploadFile(handler, url, data)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusOK {
		t.Errorf("file upload failed")
	}

	// confirm we can't reupload
	_, code, err = uploadFile(handler, url, data)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("file upload should have failed")
	}

	// confirm download matches upload
	respData, code, err := downloadFile(handler, url)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusOK {
		t.Errorf("expecting http status OK")
	}
	if bytes.Compare(data, respData) != 0 {
		t.Errorf("download differs from upload")
	}
}

func TestUploadDownload(t *testing.T) {
	os.RemoveAll(testRootDir)
	defer os.RemoveAll(testRootDir)

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
