package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const apiURL = "https://api.exchangerate-api.com/v4/latest/USD"

type ExchangeRateResponse struct {
	Rates map[string]float64 `json:"rates"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func fetchExchangeRate() (float64, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to fetch exchange rate: %s", resp.Status)
	}

	var result ExchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	rate, ok := result.Rates["BRL"]
	if !ok {
		return 0, fmt.Errorf("BRL rate not found")
	}

	return rate, nil
}

func rateUpdater(conn *websocket.Conn) {
	var lastRate float64
	for {
		rate, err := fetchExchangeRate()
		if err != nil {
			log.Println("Error fetching exchange rate:", err)
			return
		}

		if rate != lastRate {
			if err := conn.WriteJSON(map[string]float64{"rate": rate}); err != nil {
				log.Println("Error writing to WebSocket:", err)
				return
			}
			lastRate = rate
		}

		time.Sleep(1 * time.Minute) // Check for updates every minute
	}
}

func exchangeRateHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open WebSocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	rateUpdater(conn)
}

func main() {
	http.HandleFunc("/rate", exchangeRateHandler)
	http.Handle("/", http.FileServer(http.Dir(".")))
	fmt.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
