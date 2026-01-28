package moolah

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

func MakeInvoice(reqData struct {
	Amount   float64 `json:"amount"`
	Uid      int64   `json:"uid"`
	VbMethod string  `json:"vbMethod"`
	Data     string  `json:"data"`
}) (string, error) {

	// Prepare request to Crypto-Pay API
	payload := struct {
		CurrencyType string  `json:"currency_type"`
		Amount       float64 `json:"amount"`
		Asset        string  `json:"fiat"`
		Payload      string  `json:"payload"`
	}{
		CurrencyType: "fiat",
		Amount:  reqData.Amount,
		Asset:   "RUB",
		Payload: fmt.Sprintf("uid:%d", reqData.Uid),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "Error encoding payload", err
	}

	// Create request to Crypto-Pay API
	cryptoBotReq, err := http.NewRequest("POST", "https://pay.crypt.bot/api/createInvoice", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "Error creating request", err
	}

	// auth headers
	apiKey := os.Getenv("CRYPTO_BOT_APIKEY")
	if apiKey == "" {
		return "CRYPTO_BOT_APIKEY not set", errors.New("CRYPTO_BOT_APIKEY not set")
	}

	cryptoBotReq.Header.Set("Crypto-Pay-API-Token", apiKey)
	cryptoBotReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	cryptoBotResp, err := client.Do(cryptoBotReq)
	if err != nil {
		return "Error sending request to Crypto-Pay API", errors.New("Error sending request to Crypto-Pay API")
	}
	defer cryptoBotResp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(cryptoBotResp.Body)
	if err != nil {
		return "Error reading response", err
	}

	return string(respBody), nil
}
