package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	fmt.Printf("\nReceived API request: %+v\n", req)

	if req.VbMethod == "createCryptoExchange" {
		cryptoAmount, err := moolah.Cmc_getPriceRub(float64(req.Amount), req.Data)

		returnCode := http.StatusOK
		if err != nil {
			returnCode = http.StatusInternalServerError
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(returnCode)
		w.Write([]byte(strconv.FormatFloat(cryptoAmount, 'f', -1, 64)))

		return
	}

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
	signature := strings.TrimSpace(r.Header.Get("crypto-pay-api-signature"))
	bodyBytes, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		fmt.Printf("cannot read body: %s", err)
		return
	}

	token := os.Getenv("CRYPTO_BOT_APIKEY")
	if token == "" {
		fmt.Print("CRYPTO_BOT_APIKEY not set")
		return
	}
	secret := sha256.Sum256([]byte(token)) 
	mac := hmac.New(sha256.New, secret[:])
	mac.Write(bodyBytes)
	expectedRaw := mac.Sum(nil)

	if strings.HasPrefix(strings.ToLower(signature), "sha256=") {
		signature = signature[len("sha256="):]
	}
	decoded, derr := hex.DecodeString(strings.TrimSpace(signature))
	if derr != nil {
		fmt.Printf("invalid signature header (not hex): %v\n", derr)
		return
	}
	if !hmac.Equal(decoded, expectedRaw) {
		fmt.Printf("crypto hook signature mismatch: got=%s expected_prefix=%x...\n", signature, expectedRaw[:6])
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var crptHook cryptoHook
	err = json.NewDecoder(r.Body).Decode(&crptHook)
	if err != nil {
		fmt.Printf("Fail to decode body to json obj.BAD JSON")
		return
	}

	updType := crptHook.UpdateType
	if updType != "invoice_paid" {
		fmt.Println("Unknown upd type WebHooked. Type: " + updType)
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
