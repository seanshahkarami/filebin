package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func downloadFile(url string) ([]byte, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func uploadFile(url string, body []byte) ([]byte, int, error) {
	resp, err := http.Post(url, "", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func TestUploadDownload(t *testing.T) {
	root := "dataserver_test"
	os.RemoveAll(root)

	handler := http.NewServeMux()
	handler.Handle("/data/", http.StripPrefix("/data/", DataServer(root)))

	server := http.Server{
		Addr:         "127.0.0.1:10000",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      handler,
	}

	go server.ListenAndServe()
	defer server.Shutdown(context.TODO())

	if _, code, err := downloadFile("http://127.0.0.1:10000/data/file1"); err != nil || code != http.StatusNotFound {
		if err != nil {
			t.Error(err)
		} else {
			t.Errorf("expected no file")
		}
	}

	testData := []byte("here's some test data")

	if _, code, err := uploadFile("http://127.0.0.1:10000/data/file1", testData); err != nil || code != http.StatusOK {
		if err != nil {
			t.Error(err)
		} else {
			t.Errorf("file upload failed")
		}
	}

	data, code, err := downloadFile("http://127.0.0.1:10000/data/file1")
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusOK {
		t.Errorf("expecting http status OK")
	}
	if bytes.Compare(data, testData) != 0 {
		t.Errorf("downlaod differs from upload")
	}

	os.RemoveAll(root)
}
