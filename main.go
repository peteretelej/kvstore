/*

 */
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Configs used by cli client to get and set values
var (
	store = os.Getenv("KVSTORE")
	cred  = os.Getenv("KVCRED")
)

func main() {
	var (
		// Server flags
		listen = flag.String("listen", ":8080", "http listen address")
		creds  = flag.String("creds", "creds.json", "creds json file containing client credentials")

		// Client flags
		get = flag.Bool("get", false, "get from kvstore")
		set = flag.Bool("set", false, "set key value to kvstore")
		k   = flag.String("k", "", "kvstore key")
		v   = flag.String("v", "", "kvstore value")
	)
	flag.Parse()
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if !*get && !*set {
		serve(*listen, *creds)
		return
	}

	if store == "" || cred == "" {
		fmt.Fprintf(os.Stderr, "kvstore client cannot get/set without $KVSTORE and $KVCRED in env\n")
		os.Exit(1)
	}

	cl, err := NewClient(store, cred)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to connect to store %s: %v\n", store, err)
		os.Exit(1)
	}
	if *k == "" {
		fmt.Fprintln(os.Stderr, "kvstore cannot get/set without key -k")
		os.Exit(1)
	}

	if *get {
		v, err := cl.Get(*k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
		fmt.Fprintln(os.Stdout, v)
		return
	}
	if *set {
		if err := cl.Set(*k, *v); err != nil {
			fmt.Fprintf(os.Stderr, "unable to get from store: %v\n", err)
		}
	}
}

func serve(listenAddr, credsFile string) {
	h, err := newHandler(credsFile)
	if err != nil {
		fmt.Printf("init failed: %v\n", err)
		os.Exit(1)
	}
	http.Handle("/", h)
	svr := &http.Server{
		Addr:           listenAddr,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	fmt.Printf("launching http server on %s\n", listenAddr)
	if err := svr.ListenAndServe(); err != nil {
		fmt.Printf("server crashed: %s\n", err)
		os.Exit(1)
	}
}

type handler struct {
	credsMap map[string]bool
}

func newHandler(credsFile string) (*handler, error) {
	dat, err := ioutil.ReadFile(credsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s for client credentials", credsFile)
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
	cred := r.FormValue("cred")
	if cred == "" {
		fmt.Fprint(w, "")
		return
	}
	if !h.validCred(cred) {
		http.Error(w, "invalid cred", http.StatusUnauthorized)
		return
	}
	if r.FormValue("k") == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
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
	v, err := tempStore.get(r.FormValue("k"))
	if err != nil {
		http.Error(w, "no such key in store", http.StatusNotFound)
		return
	}
	fmt.Fprint(w, v)
}
func (h *handler) put(w http.ResponseWriter, r *http.Request) {
	k, v := strings.TrimSpace(r.FormValue("k")), r.FormValue("v")
	if k == "" {
		http.Error(w, "key cannot be empty space", http.StatusBadRequest)
		return
	}
	tempStore.set(k, v)
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

func (kv *kvStore) set(key, value string) {
	kv.mu.Lock()
	kv.kv[key] = value
	kv.mu.Unlock()
}
func (kv *kvStore) get(key string) (string, error) {
	kv.mu.RLock()
	v, ok := kv.kv[key]
	kv.mu.RUnlock()
	if !ok {
		return "", errors.New("miss")
	}
	return v, nil

}

// Client defines the structure of a kvstore cli client
type Client struct {
	Store  *url.URL     // url containing cred query param
	client *http.Client // the http.Client to use
}

// NewClient verifies the credential and server and returns a client for use
func NewClient(store, cred string) (*Client, error) {
	u, err := url.Parse(store)
	if err != nil {
		return nil, fmt.Errorf("invalid store url: %v", err)
	}
	q := u.Query()
	q.Set("cred", cred)
	u.RawQuery = q.Encode()
	cli := &Client{Store: u}
	cli.client = &http.Client{Timeout: time.Second}
	req, err := http.NewRequest("GET", cli.Store.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("invalid store submitted: %s", store)
	}
	resp, err := cli.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot reach store endpoint %s", store)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Print(err)
		}
	}()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("invalid -cred for store %s", store)
	}
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("invalid response from store")
	}
	if !strings.Contains(string(dat), "missing key") {
		return nil, fmt.Errorf("invalid response from store: %s", dat)
	}
	return cli, nil
}

// Get is used by the cli client to fetch the value of a key
func (cli *Client) Get(key string) (string, error) {
	q := cli.Store.Query()
	q.Add("k", key)
	cli.Store.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", cli.Store.String(), nil)
	if err != nil {
		return "", fmt.Errorf("unable to get from store: %v", err)
	}
	resp, err := cli.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("invalid response from store: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Print(err)
		}
	}()
	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("not in store")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response from store not OK: %s", http.StatusText(resp.StatusCode))
	}
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("invalid response from the store:%v", err)
	}
	return string(dat), nil
}

// Set is used by the cli client to set the value of a key
func (cli *Client) Set(key, value string) error {
	q := cli.Store.Query()
	q.Add("k", key)
	q.Add("v", value)
	cli.Store.RawQuery = q.Encode()
	req, err := http.NewRequest("PUT", cli.Store.String(), nil)
	if err != nil {
		return fmt.Errorf("unable to put to store: %v", err)
	}
	resp, err := cli.client.Do(req)
	if err != nil {
		return fmt.Errorf("invalid response from store: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Print(err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response from store not OK: %s", http.StatusText(resp.StatusCode))
	}
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("invalid response from store: %v", err)
	}
	want := fmt.Sprintf("%s: %s", key, value)
	if !bytes.Equal([]byte(want), dat) {
		return fmt.Errorf("unable to set key value in store")
	}
	return nil
}
