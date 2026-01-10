package main

import (
	"fmt"
	"io"
	"net/http"
)

func echoHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading body", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(body)
}

func main() {
	http.HandleFunc("/vb-api", echoHandler)
	fmt.Println("Server starting on :8001")
	http.ListenAndServe(":8001", nil)
}
