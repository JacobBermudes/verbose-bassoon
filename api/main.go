package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
	"verbose-bassoon/api/moolah"

	"github.com/redis/go-redis/v9"
)

var rdbpass = os.Getenv("REDIS_PASS")
var ctx = context.Background()

var acc_db = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	DB:       15,
	Password: rdbpass,
})

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
		Data     string  `json:"data"`
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

	key := "user:" + strconv.FormatInt(req.Uid, 10)

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

	if req.VbMethod == "accountInit" {

		regTime := time.Now().Format("2006-01-02")
		cid, _ := strconv.ParseInt(req.Data, 10, 64)

		userExist, err := acc_db.HExists(ctx, key, "created_at").Result()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if userExist {
			w.WriteHeader(http.StatusOK)
			return
		}

		err = acc_db.HSet(ctx, key, "created_at", regTime, "cid", cid, "balance", 0).Err()
		if err != nil {
			fmt.Println(key + " didnt init")
		}
		w.WriteHeader(http.StatusOK)
		fmt.Println(key + " created!")
		return
	}

	if req.VbMethod == "getBalance" {
		
		balance, err := acc_db.HGet(ctx, key, "balance").Result()
		if err != nil {
			fmt.Printf("Something goes wrong while gettin balance")
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(balance))
		
		return
	}

	http.Error(w, "Fuck off", http.StatusBadRequest)

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
