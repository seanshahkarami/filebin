package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
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

func testUploadDownload(t *testing.T, url string, data []byte) {
	_, code, err := downloadFile(url)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusNotFound {
		t.Errorf("file should not exist yet")
	}

	_, code, err = uploadFile(url, data)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusOK {
		t.Errorf("file upload failed")
	}

	// confirm we can't reupload
	_, code, err = uploadFile(url, data)
	if err != nil {
		t.Error(err)
	}
	if code != http.StatusBadRequest {
		t.Errorf("file upload should have failed")
	}

	// confirm download matches upload
	respData, code, err := downloadFile(url)
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

	fmt.Printf("\ntesting many small file uploads\n")
	for i := 0; i < 100; i++ {
		url := fmt.Sprintf("http://127.0.0.1:10000/data/file%d", i)
		data := []byte(fmt.Sprintf("here's some test data - test %d", i))
		testUploadDownload(t, url, data)
	}

	fmt.Printf("\ntesting large file upload\n")
	url := "http://127.0.0.1:10000/data/largefile"
	data := make([]byte, 32*1024*1024)
	rand.Read(data)
	testUploadDownload(t, url, data)

	os.RemoveAll(root)
}
