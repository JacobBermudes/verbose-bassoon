package moolah

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

var CMC_KEY = os.Getenv("COIN_MC_APIKEY")

type cmc_rate_r struct {
	Data []rates `json:"data"`
}
type rates struct {
	ID    float64              `json:"id"`
	Quote map[string]coin_rate `json:"quote"`
}
type coin_rate struct {
	Price      float64 `json:"price"`
	LastUpdate string  `json:"last_updated"`
}

func Cmc_getPriceRub(amount float64, coin string) (float64, error) {
	client := &http.Client{}

	params := url.Values{}
	params.Add("amount", fmt.Sprintf("%.2f", amount))
	params.Add("symbol", "RUB")
	params.Add("convert", coin)

	req, err := http.NewRequest("GET", "https://pro-api.coinmarketcap.com/v2/tools/price-conversion", nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return 0, err
	}

	req.Header.Set("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", CMC_KEY)
	req.URL.RawQuery = params.Encode()

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request to server. Error: %v\n", err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("Non-OK HTTP status: %s. Response body: %s\n", resp.Status, string(bodyBytes))
		return 0, fmt.Errorf("non-OK HTTP status: %s", resp.Status)
	}

	var result cmc_rate_r
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Fail unmarshal JSON: %v\n", err)
		return 0, err
	}

	cryptoAmount := result.Data[0].Quote[coin].Price
	fmt.Printf("Get rate from CCM. Amount: %f\n", cryptoAmount)

	return cryptoAmount, nil
}
