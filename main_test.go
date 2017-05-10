package main

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	_, err := newHandler("")
	if err == nil {
		t.Errorf("newHandler test failed, got %v, want error", err)
	}
	if !strings.Contains(err.Error(), "unable to read") {
		t.Errorf("newHandler got '%v', want 'unable to read'", err)
	}
	tmpjson, err := ioutil.TempFile("", "cred.json")
	if err != nil {
		t.Fatalf("unable to create tmp json file for testing: %v", err)
	}
	cred := tmpjson.Name()
	if err := tmpjson.Close(); err != nil {
		t.Fatalf("unable to close tmp json file for usage in tests: %v", err)
	}
	defer func() {
		if err := os.Remove(cred); err != nil {
			t.Fatalf("unable to delete test json file: %v", err)
		}
	}()
	// blank cred file
	_, err = newHandler(cred)
	if err == nil {
		t.Errorf("newHandler test failed, got %v, want error", err)
	}
	if !strings.Contains(err.Error(), "unable to parse") {
		t.Errorf("newHandler want 'unable to parse', got '%v'", err)
	}

	err = ioutil.WriteFile(cred, []byte(`[]`), 0644)
	if err != nil {
		t.Fatalf("unable to write test cred json file: %v", err)
	}
	_, err = newHandler(cred)
	if err == nil {
		t.Errorf("newHandler test failed, got %v, want error", err)
	}
	if !strings.Contains(err.Error(), "no creds found") {
		t.Errorf("newHandler want 'no creds found', got '%v'", err)
	}

	err = ioutil.WriteFile(cred, []byte(`["testcredential"]`), 0644)
	if err != nil {
		t.Fatalf("unable to write test cred json file: %v", err)
	}
	h, err := newHandler(cred)
	if err != nil {
		t.Errorf("newHandler test failed, got %v, want %v", err, nil)
	}
	testMap := map[string]bool{"testcredential": true}
	if !reflect.DeepEqual(h.credsMap, testMap) {
		t.Errorf("newHandler test failed, got %v, want %v", h.credsMap, testMap)
	}
}

func TestHandlerServeHTTP(t *testing.T) {
	tmpjson, err := ioutil.TempFile("", "cred.json")
	if err != nil {
		t.Fatalf("unable to create tmp json file for testing: %v", err)
	}
	cred := tmpjson.Name()
	if err := tmpjson.Close(); err != nil {
		t.Fatalf("unable to close tmp json file for usage in tests: %v", err)
	}
	defer func() {
		if err := os.Remove(cred); err != nil {
			t.Fatalf("unable to delete test json file: %v", err)
		}
	}()

	err = ioutil.WriteFile(cred, []byte(`["testcredential"]`), 0644)
	if err != nil {
		t.Fatalf("unable to write test cred json file: %v", err)
	}
	h, err := newHandler(cred)
	if err != nil {
		t.Errorf("newHandler test failed, got %v, want %v", err, nil)
	}

	type httptestfields struct {
		method string
		url    string
		status int
		body   string
	}
	tests := []httptestfields{
		{"GET", "/hello", http.StatusBadRequest, "invalid request"},
		{"GET", "/", http.StatusOK, ""},
		{"GET", "/?cred=badcredential", http.StatusUnauthorized, "invalid cred"},
		{"GET", "/?cred=testcredential", http.StatusBadRequest, "missing key"},
		{"GET", "/?cred=testcredential&k=hello", http.StatusNotFound, "no such key"},
		{"PUT", "/?cred=testcredential&k=hello&v=world", http.StatusOK, "hello: world"},
		{"PUT", "/?cred=testcredential&v=world", http.StatusBadRequest, "missing key"},
		{"PUT", "/?cred=testcredential&k=%20&v=world", http.StatusBadRequest, "key cannot be empty space"},
		{"PUT", "/?cred=testcredential&k=hello", http.StatusOK, "hello: "},
		{"PUT", "/?cred=testcredential&k=hello&v=gopher", http.StatusOK, "hello: gopher"},
		{"GET", "/?cred=testcredential&k=hello", http.StatusOK, "gopher"},
	}

	for _, v := range tests {
		req, err := http.NewRequest(v.method, v.url, nil)
		if err != nil {
			t.Fatalf("unable to generate test request: %v", err)
		}
		rr := httptest.NewRecorder() // responserecoder
		func() {
			h.ServeHTTP(rr, req)
			if rr.Code != v.status {
				t.Errorf("handler ServeHTTP %s %s status code got %d, want %d", v.method, v.url, rr.Code, v.status)
			}
			if !bytes.Contains(rr.Body.Bytes(), []byte(v.body)) {
				t.Errorf("handler ServeHTTP %s %s got '%s', want '%s'", v.method, v.url, rr.Body.Bytes(), v.body)
			}
		}()
	}
}
