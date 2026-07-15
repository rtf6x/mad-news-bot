package currency

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const currenciesURL = "https://www.cbr-xml-daily.ru/daily_json.js"

var randomCurrencies = []string{
	"AUD", "AZN", "AMD", "BYN", "BGN", "BRL", "HUF", "HKD", "DKK", "INR", "KZT", "CAD", "KGS", "CNY", "MDL", "NOK",
	"RON", "XDR", "SGD", "TJS", "TRY", "TMT", "UZS", "UAH", "CZK", "SEK", "ZAR", "KRW",
}

type cbrResponse struct {
	Date         string `json:"Date"`
	PreviousDate string `json:"PreviousDate"`
	Valute       map[string]struct {
		Name     string  `json:"Name"`
		Nominal  float64 `json:"Nominal"`
		Value    float64 `json:"Value"`
		Previous float64 `json:"Previous"`
	} `json:"Valute"`
}

func Format(ctx context.Context, currencyKey string) (string, error) {
	data, err := fetch(ctx)
	if err != nil {
		return "", err
	}
	if data.Valute == nil ||
		data.Valute["USD"].Value == 0 ||
		data.Valute["EUR"].Value == 0 {
		return "Не могу получить данные с cbr-xml-daily ;(", nil
	}

	updated := formatCBRDate(data.Date)
	prevDate := formatCBRDate(data.PreviousDate)

	var currencies string
	if currencyKey != "" {
		currencies = getCurrencyData(data, currencyKey)
	} else {
		randomKey := randomCurrencies[rand.Intn(len(randomCurrencies))]
		currencies = strings.Join([]string{
			getCurrencyData(data, "USD"),
			getCurrencyData(data, "EUR"),
			getCurrencyData(data, "RSD"),
			getCurrencyData(data, "GBP"),
			getCurrencyData(data, "GEL"),
			getCurrencyData(data, "CHF"),
			getCurrencyData(data, "PLN"),
			getCurrencyData(data, randomKey),
		}, "\n")
	}

	return fmt.Sprintf("[Обновлено: %s | Предыдущие данные: %s]\n\n%s\n\n(Источник: cbr-xml-daily.ru)\n",
		updated, prevDate, currencies), nil
}

func fetch(ctx context.Context) (*cbrResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, currenciesURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data cbrResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func getCurrencyData(data *cbrResponse, currencyKey string) string {
	val, ok := data.Valute[currencyKey]
	if !ok {
		return fmt.Sprintf("[%s] Нет данных по валюте!", currencyKey)
	}
	change := val.Value - val.Previous
	if change < 0 {
		change = -change
	}
	isRise := "сдулось"
	if val.Value > val.Previous {
		isRise = "опухло"
	}
	changed := "не изменилось"
	if change != 0 {
		changed = fmt.Sprintf("%s на %.2f", isRise, change)
	}
	name := val.Name
	if val.Nominal > 1 {
		name = fmt.Sprintf("%.0f %s", val.Nominal, val.Name)
	}
	return fmt.Sprintf("[%s][%s] %v (%s)", currencyKey, name, val.Value, changed)
}

func formatCBRDate(raw string) string {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	loc, _ := time.LoadLocation("Europe/Moscow")
	return t.In(loc).Format("02.01.2006 (15:04)")
}
