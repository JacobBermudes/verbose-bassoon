package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"verbose-bassoon/api/moolah"
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

	if req.VbMethod == "createInvoice" {
		invoiceLink, err := moolah.MakeInvoice(req)

		returnCode := http.StatusOK; if err != nil { returnCode = http.StatusInternalServerError }

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(returnCode)
		w.Write([]byte(invoiceLink))

		return
	}

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
