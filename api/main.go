package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

func apiHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Amount   float64 `json:"amount"`
		Uid      int64   `json:"uid"`
		VbMethod string  `json:"vbMethod"`
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received API request: %+v\n", req)

	if req.VbMethod != "createInvoice" {
		http.Error(w, "Unsupported vbMethod", http.StatusBadRequest)
		return
	}

	// Prepare request to Crypto-Pay API
	payload := struct {
		Amount  float64 `json:"amount"`
		Asset   string  `json:"asset"`
		Payload string  `json:"payload"`
	}{
		Amount:  req.Amount,
		Asset:   "TON",
		Payload: fmt.Sprintf("uid:%d", req.Uid),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "Error encoding payload", http.StatusInternalServerError)
		return
	}

	// Create request to Crypto-Pay API
	cryptoBotReq, err := http.NewRequest("POST", "https://pay.crypt.bot/api/createInvoice", bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	// Set headers
	cryptoBotReq.Header.Set("Content-Type", "application/json")
	apiKey := os.Getenv("CRYPTO_BOT_APIKEY")
	if apiKey == "" {
		http.Error(w, "CRYPTO_BOT_APIKEY not set", http.StatusInternalServerError)
		return
	}
	cryptoBotReq.Header.Set("Crypto-Pay-API-Token", apiKey)

	// Send request
	client := &http.Client{}
	cryptoBotResp, err := client.Do(cryptoBotReq)
	if err != nil {
		http.Error(w, "Error sending request to Crypto-Pay API", http.StatusInternalServerError)
		return
	}
	defer cryptoBotResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(cryptoBotResp.Body)
	if err != nil {
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		return
	}

	// Send response back
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cryptoBotResp.StatusCode)
	w.Write(respBody)
}

func cryptoHookHandler(w http.ResponseWriter, r *http.Request) {
	var invoice struct {
		InvoiceID int64  `json:"invoice_id"`
		Status    string `json:"status"`
	}
	err := json.NewDecoder(r.Body).Decode(&invoice)
	if err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}
	fmt.Printf("Received crypto hook: InvoiceID=%d, Status=%s\n", invoice.InvoiceID, invoice.Status)
	w.WriteHeader(http.StatusOK)
}

func makeInvoiceHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Amount float64 `json:"amount"`
		UserID int64   `json:"user_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Error decoding JSON", http.StatusBadRequest)
		return
	}

	invoiceID := req.UserID*1000 + int64(req.Amount*100)
	resp := struct {
		InvoiceID int64  `json:"invoice_id"`
		PayURL    string `json:"pay_url"`
	}{
		InvoiceID: invoiceID,
		PayURL:    fmt.Sprintf("https://crypto.payments.example/invoice/%d", invoiceID),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func main() {
	http.HandleFunc("/vb-api", echoHandler)
	http.HandleFunc("/vb-api/balance", balanceHandler)
	http.HandleFunc("/vb-api/crypto-hook", cryptoHookHandler)
	http.HandleFunc("/vb-api/v1", apiHandler)
	fmt.Println("Server starting on :8001")
	http.ListenAndServe(":8001", nil)
}
