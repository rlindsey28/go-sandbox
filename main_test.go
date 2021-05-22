package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_homeHandler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(homeHandler))

	defer ts.Close()

	r, err := http.Get(ts.URL)
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}

	result, err	 := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}
	defer func() {
		log.Println("defer func")
	}()

	if !strings.Contains(string(result), "container") {
		t.Logf("#{result}")
		t.Fatal("not in container")
	}
}
