package main

import (
	"encoding/json"
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

func balanceHandler(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	var balance int64 = 1122 + int64(len(uid))*33
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

func main() {
	http.HandleFunc("/vb-api", echoHandler)
	http.HandleFunc("/vb-api/balance", balanceHandler)
	fmt.Println("Server starting on :8001")
	http.ListenAndServe(":8001", nil)
}
