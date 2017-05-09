package main

import (
	"io/ioutil"
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
		t.Errorf("newHandler want 'unable to read', got '%v'", err)
	}
	tmpjson, err := ioutil.TempFile("", "cred.json")
	if err != nil {
		t.Fatalf("unable to create tmp json file for testing: %v", err)
	}
	cred := tmpjson.Name()
	if err := tmpjson.Close(); err != nil {
		t.Fatalf("unable to close tmp json file for usage in tests: %v", err)
	}
	defer os.Remove(cred)
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
