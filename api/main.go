package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"verbose-bassoon/api/moolah"
)

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

	if req.VbMethod == "createCryptoInvoice" {
		invoiceLink, err := moolah.MakeInvoice(req)

		returnCode := http.StatusOK
		if err != nil {
			returnCode = http.StatusInternalServerError
		}

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

func main() {
	http.HandleFunc("/vb-api/balance", balanceHandler)
	http.HandleFunc("/vb-api/crypto-hook", cryptoHookHandler)
	http.HandleFunc("/vb-api/v1", apiHandler)
	fmt.Println("Server starting on :8001")
	http.ListenAndServe(":8001", nil)
}
