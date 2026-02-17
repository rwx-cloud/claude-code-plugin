package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "api ok")
	})
	http.ListenAndServe(":8080", nil)
}
