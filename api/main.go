package main

import (
	"bytes"
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

type cryptoHook struct {
	UpdateType string            `json:"update_type"`
	Payload    cryptoHookPayload `json:"payload"`
}

type cryptoHookPayload struct {
	HookedPayload string `json:"payload"`
	Amount        string `json:"amount"`
}

func apiHandler(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Amount   int64  `json:"amount"`
		Uid      int64  `json:"uid"`
		VbMethod string `json:"vbMethod"`
		Data     string `json:"data"`
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
	var crptHook cryptoHook
	err := json.NewDecoder(r.Body).Decode(&crptHook)
	if err != nil {
		fmt.Print("Fail to decode body to json obj.BAD JSON")
		return
	}

	updType := crptHook.UpdateType
	if updType != "invoice_paidReceived" {
		fmt.Print("Unknown upd type WebHooked. Type: " + updType)
		w.WriteHeader(http.StatusOK)
		return
	}

	paidAmount, _ := strconv.ParseInt(crptHook.Payload.Amount, 10, 64)
	paidUid := crptHook.Payload.HookedPayload

	newBalance, err := acc_db.HIncrBy(ctx, paidUid, "balance", paidAmount).Result()
	if err != nil {
		fmt.Printf("FAILED TO INCR BALANCE")
	}

	var notifyReq struct {
		Cid  string `json:"cid"`
		Text string `json:"text"`
	}
	notifyReq.Cid, _ = acc_db.HGet(ctx, paidUid, "cid").Result()
	notifyReq.Text = fmt.Sprintf("Баланс успешно пополнен на %s рублей!\nБаланс: %d", crptHook.Payload.Amount, newBalance)
	notifyPayload, err := json.Marshal(notifyReq)
	if err != nil {
		fmt.Println("Get balance request JSON marshal error")
	}

	http.Post("http://127.0.0.1:8011/vb-notify", "application/json", bytes.NewBuffer(notifyPayload))

	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/vb-api/crypto-hook", cryptoHookHandler)
	http.HandleFunc("/vb-api/v1", apiHandler)
	fmt.Println("Server starting on :8001")
	http.ListenAndServe(":8001", nil)
}
