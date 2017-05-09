package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	listen = flag.String("listen", ":9112", "http listen address")
	creds  = flag.String("creds", ".creds.json", "creds json file containing client credentials")
)

func main() {
	h, err := newHandler(*creds)
	if err != nil {
		fmt.Printf("init failed: %v\n", err)
		os.Exit(1)
	}
	http.Handle("/", h)
	svr := &http.Server{
		Addr:           *listen,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	if err := svr.ListenAndServe(); err != nil {
		fmt.Printf("server crashed: %s\n", err)
		os.Exit(1)
		return
	}
}

type handler struct {
	credsMap map[string]bool
}

func newHandler(credsFile string) (*handler, error) {
	dat, err := ioutil.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s for passwords", credsFile)
	}
	var creds []string
	err = json.Unmarshal(dat, &creds)
	if err != nil {
		return nil, fmt.Errorf("unable to parse %s into creds", credsFile)
	}
	h := &handler{credsMap: make(map[string]bool)}
	for _, cred := range creds {
		if cred := strings.TrimSpace(cred); cred != "" {
			h.credsMap[cred] = true
		}
	}
	if len(h.credsMap) == 0 {
		return nil, fmt.Errorf("no creds found in %s", credsFile)
	}
	return h, nil
}

// ServeHTTP is handlers implementation for serving http
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "PUT":
		h.put(w, r)
		return
	}
	http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	return

}
func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	if !h.validCred(r.URL.Query().Get("cred")) {
		http.Error(w, "invalid cred", http.StatusUnauthorized)
		return
	}
	v, err := get(r.Form.Get("k"))
	if err != nil {
		fmt.Fprint(w, "no such key in store")
		return
	}
	fmt.Fprint(w, v)
}
func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "unable to parse post request", http.StatusBadRequest)
		return
	}
	if !h.validCred(r.Form.Get("cred")) {
		http.Error(w, "invalid cred", http.StatusUnauthorized)
		return
	}
	k, v := r.Form.Get("k"), r.Form.Get("v")
	k, v = strings.TrimSpace(k), strings.TrimSpace(v)
	if k == "" {
		http.Error(w, "key cannot be empty", http.StatusBadRequest)
		return
	}
	set(k, v)
	fmt.Fprintf(w, "%s: %s", k, v)
}

func (h *handler) validCred(cred string) bool {
	cred = strings.TrimSpace(cred)
	_, ok := h.credsMap[cred]
	return ok
}

type kvStore struct {
	mu sync.RWMutex // protects the key value map below
	kv map[string]string
}

var tempStore = &kvStore{kv: make(map[string]string)}

func set(key, value string) {
	tempStore.mu.Lock()
	tempStore.kv[key] = value
	tempStore.mu.Unlock()
}
func get(key string) (string, error) {
	tempStore.mu.RLock()
	v, ok := tempStore.kv[key]
	tempStore.mu.RUnlock()
	if !ok {
		return "", errors.New("miss")
	}
	return v, nil

}
