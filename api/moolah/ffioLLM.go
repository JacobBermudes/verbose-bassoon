// Package moolah — простой клиент для интеграции с ff.io (создание адресов для депозита).
// Файл: /home/mickey/src/verbose-bassoon/api/moolah/ffio.go
package moolah

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Простой, безопасный и легко адаптируемый HTTP-клиент для ff.io.
// Код не привязан жестко к конкретному полю API — при необходимости
// можно изменить путь или полезную нагрузку в опциях.

const (
	DefaultBaseURL      = "https://ff.io"         // поменяйте, если нужно
	DefaultCreatePath   = "/api/v1/deposit"       // пример пути — подправьте по реальной документации
	DefaultHTTPTimeout  = 15 * time.Second
	maxErrorBodyPreview = 1024
)

// ClientOption опция для NewClient.
type ClientOption func(*Client)

// Client контекст для запросов к ff.io.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	createPath string
}

// NewClient создаёт клиента. apiKey может быть пустым (если сервис не требует).
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	u, err := url.Parse(DefaultBaseURL)
	if err != nil {
		return nil, err
	}
	c := &Client{
		baseURL:    u,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: DefaultHTTPTimeout},
		createPath: DefaultCreatePath,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// WithBaseURL переопределяет базовый URL (для тестов или если документация требует другой).
func WithBaseURL(raw string) ClientOption {
	return func(c *Client) {
		if u, err := url.Parse(raw); err == nil {
			c.baseURL = u
		}
	}
}

// WithHTTPClient задаёт кастомный http.Client.
func WithHTTPClient(h *http.Client) ClientOption {
	return func(c *Client) {
		if h != nil {
			c.httpClient = h
		}
	}
}

// WithCreatePath позволяет настроить путь создания адреса (если документация ff.io иная).
func WithCreatePath(p string) ClientOption {
	return func(c *Client) {
		if p != "" {
			c.createPath = p
		}
	}
}

// DepositRequest тело запроса для создания адреса депозита.
// Поля подстраиваемые — подправьте названия/структуру по реальной API-документации ff.io.
type DepositRequest struct {
	Currency  string            `json:"currency"`            // монета, в которой пришлют крипту (например "btc", "eth" и т.д.)
	ForwardTo string            `json:"forward_to,omitempty"`// адрес куда пересылать (TON-кошелек)
	Metadata  map[string]string `json:"metadata,omitempty"`  // доп. данные (order id и т.п.)
	// если ff.io требует иные поля — добавьте тут
}

// DepositResponse ожидаемая структура ответа.
// Подправьте под реальную документацию (имена полей могут отличаться).
type DepositResponse struct {
	ID        string     `json:"id,omitempty"`
	Currency  string     `json:"currency,omitempty"`
	Address   string     `json:"address,omitempty"`
	Tag       string     `json:"tag,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Status    string     `json:"status,omitempty"`
	ForwardTo string     `json:"forward_to,omitempty"`
	Raw       json.RawMessage
}

// APIError структурированная ошибка от ff.io (если сервер возвращает JSON error).
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ffio: status=%d message=%s body=%s", e.StatusCode, e.Message, e.Body)
}

// CreateDepositAddress создаёт адрес для приёма указанной валюты.
// forwardTON можно передавать — тогда конечный приём будет на TON-адрес.
// Валидатор TON-адреса базовый — адаптируйте под ваш формат (EQ... или 0:<hex>).
func (c *Client) CreateDepositAddress(ctx context.Context, req DepositRequest) (*DepositResponse, error) {
	// Валидация входа
	req.Currency = strings.TrimSpace(req.Currency)
	if req.Currency == "" {
		return nil, errors.New("currency is required")
	}
	if req.ForwardTo != "" && !isLikelyTONAddress(req.ForwardTo) {
		return nil, fmt.Errorf("forward_to does not look like a TON address: %q", req.ForwardTo)
	}

	// Собираем URL
	ep, err := c.baseURL.Parse(c.createPath)
	if err != nil {
		return nil, err
	}

	// Сериализация тела
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Запрос
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MiB limit

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Попытаться распарсить структурированную ошибку
		apiMsg := extractErrorMessage(respBody)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    apiMsg,
			Body:       preview(respBody),
		}
	}

	var out DepositResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		// вернуть сырое тело для диагностики
		return nil, fmt.Errorf("failed to decode ff.io response: %w (body=%s)", err, preview(respBody))
	}
	out.Raw = json.RawMessage(respBody)
	// Небольшая дополнительная валидация: убедиться, что мы получили адрес
	if out.Address == "" && out.Tag == "" {
		return &out, fmt.Errorf("ffio: response missing address (raw=%s)", preview(respBody))
	}
	return &out, nil
}

// CreateDepositForTON — удобный wrapper: принимает валюту и TON-адрес (строгая проверка TON).
func (c *Client) CreateDepositForTON(ctx context.Context, currency, tonAddr string, metadata map[string]string) (*DepositResponse, error) {
	if !isLikelyTONAddress(tonAddr) {
		return nil, fmt.Errorf("invalid TON address: %q", tonAddr)
	}
	req := DepositRequest{
		Currency:  currency,
		ForwardTo: tonAddr,
		Metadata:  metadata,
	}
	return c.CreateDepositAddress(ctx, req)
}

// isLikelyTONAddress простая эвристика проверки TON-адреса.
// Поддерживает форматы: EQ... (user-friendly) и 0:<hex> (raw).
func isLikelyTONAddress(s string) bool {
	if s == "" {
		return false
	}
	// EQ... адреса обычно base64-url-ish, начинаются с "EQ" (тон-кошелки от TON Labs)
	if strings.HasPrefix(s, "EQ") && len(s) >= 48 && len(s) <= 88 {
		return true
	}
	// raw hex: 0:<64 hex chars>
	if strings.HasPrefix(s, "0:") && len(s) == 66 {
		_, err := hex.DecodeString(s[2:])
		return err == nil
	}
	// допускаем также workchain:address?checksum варианты — это базовая проверка
	return false
}

// extractErrorMessage пытается достать удобоваримое сообщение из ответа сервера.
func extractErrorMessage(body []byte) string {
	var candidate map[string]any
	if err := json.Unmarshal(body, &candidate); err == nil {
		// распространённые поля
		for _, k := range []string{"error", "message", "detail", "error_description"} {
			if v, ok := candidate[k]; ok {
				if s, ok := v.(string); ok && s != "" {
					return s
				}
			}
		}
	}
	// fallback — строка тела (обрезанная)
	return preview(body)
}

func preview(b []byte) string {
	if len(b) > maxErrorBodyPreview {
		return string(b[:maxErrorBodyPreview]) + "..."
	}
	return string(b)
}

/*
Пример использования (скоротечный):

ctx := context.Background()
cli, _ := NewClient("MY_FFIO_API_KEY", WithCreatePath("/api/v1/deposit"))

// Создать адрес, чтобы пользователь прислал BTC, но средства переслать на TON-кошелек:
resp, err := cli.CreateDepositForTON(ctx, "btc", "EQC7...your_ton...", map[string]string{"order_id":"1234"})
if err != nil { ... }
// resp.Address — адрес, куда отправлять BTC
// resp.ForwardTo — ваш TON-адрес (подтверждение)
*/

// --- Тестовый хелпер (можно использовать в unit-тестах) ---

// NewTestClient создает клиент, указывающий на тестовый http.Handler.
//func NewTestClient(handler http.Handler, apiKey string) (*Client, error) {
//	srv := &http.Server{Handler: handler}
//	// используем httptest.Server при реальных тестах; здесь — просто кастомный client
//	// но для простоты создаём http.Client с Transport, который делегирует на handler через httptest
//	ts := &http.Client{Timeout: DefaultHTTPTimeout}
//	return &Client{
//		baseURL:    &url.URL{Scheme: "http", Host: "test.ff.io"},
//		apiKey:     apiKey,
//		httpClient: ts,
//		createPath: DefaultCreatePath,
//	}, nil
//}