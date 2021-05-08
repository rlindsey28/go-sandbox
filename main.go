package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	myOS, myArch := runtime.GOOS, runtime.GOARCH
	inContainer := "inside"

	if _, err := os.Lstat("/.dockerenv"); err != nil && os.IsNotExist(err) {
		inContainer = "outside"
	}

	w.Header().Set("Content-Type", "text/plain")
	responseStatus := http.StatusOK
	w.WriteHeader(responseStatus)

	_, _ = fmt.Fprintf(w, "Hello, %s!\n", r.UserAgent())
	_, _ = fmt.Fprintf(w, "I'm running on %s/%s.\n", myOS, myArch)
	_, _ = fmt.Fprintf(w, "I'm running %s a container!\n", inContainer)

	log.Printf("Response status: %v", responseStatus)

}
func main() {
	log.Println("Starting listener")

	http.HandleFunc("/getInfo", homeHandler)
	err := http.ListenAndServe(":38000", nil)
	if err != nil {
		fmt.Println(err)
	}
}
