package autoupdater

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

var (
	oldFile = []byte("old content")
	newFile = []byte("new content")
)

func TestAutoUpdater_Apply(t *testing.T) {
	testDir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(testDir)
	targetPath := filepath.Join(testDir, "testfile")
	oldFilePath := filepath.Join(testDir, ".testfile.old")
	if err := ioutil.WriteFile(targetPath, oldFile, 0666); err != nil {
		log.Fatal(err)
	}

	autoUpdater, err := NewAutoUpdater(Options{Target: targetPath})
	if err != nil {
		t.Error(err)
	}
	err = autoUpdater.Apply(bytes.NewReader(newFile))
	if err != nil {
		t.Error(err)
	}

	buf, err := ioutil.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf, newFile) {
		t.Fatalf("File was not updated! Bytes read: %v, Bytes expected: %v", buf, newFile)
	}

	if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
		t.Fatalf("Failed to find the old file: %v", err)
	}
}

func TestAutoUpdater_handleChange_invalid(t *testing.T) {
	autoUpdater, err := NewAutoUpdater(Options{})
	if err != nil {
		t.Error(err)
	}
	invalidVersionInfo := []byte("invalid")
	_, err = autoUpdater.handleChange(invalidVersionInfo)
	if err == nil {
		t.Error("Failed to throw error on invalid version info")
	}
}

func TestAutoUpdater_handleChange_success(t *testing.T) {
	testBinary := []byte("fake softare bianry")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write(testBinary)
		if err != nil {
			t.Error(err)
		}
	}))
	autoUpdater, err := NewAutoUpdater(Options{})
	if err != nil {
		t.Error(err)
	}
	invalidVersionInfo := []byte(fmt.Sprintf(
		`{"VersionId": "0.0.2", "DownloadURL": "%s"}`, ts.URL))
	update, err := autoUpdater.handleChange(invalidVersionInfo)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(update, testBinary) {
		t.Errorf("Wrong downloaded content: %s vs %s", update, testBinary)
	}
}
