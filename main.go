package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type PointsResponse struct {
	Points int `json:"points"`
}

var receiptMap map[string]int

func ProcessReceiptHandler(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}

	// Validate the receipt
	if receipt.Retailer == "" ||
		receipt.PurchaseDate == "" ||
		receipt.PurchaseTime == "" ||
		len(receipt.Items) == 0 ||
		receipt.Total == "" {
		http.Error(w, "The receipt is invalid", http.StatusBadRequest)
		return
	}

	// Validate the items
	for _, item := range receipt.Items {
		if item.ShortDescription == "" || item.Price == "" {
			http.Error(w, "The receipt is invalid", http.StatusBadRequest)
			return
		}
	}

	// Generate a unique ID for the receipt
	receiptID := uuid.New().String()

	// Calculate the points for the receipt
	points := calculatePoints(&receipt)

	receiptMap[receiptID] = points

	// Return the ID of the receipt
	response := map[string]string{"id": receiptID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetPointsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Look up the receipt by ID
	points, found := receiptMap[id]
	if !found {
		http.Error(w, "No receipt found for that id", http.StatusNotFound)
		return
	}

	// Calculate and return the points for the receipt
	response := PointsResponse{Points: points}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func calculatePoints(receipt *Receipt) int {
	points := 0

	// Rule 1: One point for every alphanumeric character in the retailer name.
	points += len(regexp.MustCompile(`[a-zA-Z0-9]`).FindAllString(receipt.Retailer, -1))

	// Rule 2: 50 points if the total is a round dollar amount with no cents.
	totalFloat, _ := strconv.ParseFloat(receipt.Total, 64)
	if math.Mod(totalFloat, 1) == 0 {
		points += 50
	}

	// Rule 3: 25 points if the total is a multiple of 0.25.
	if math.Mod(totalFloat, 0.25) == 0 {
		points += 25
	}

	// Rule 4: 5 points for every two items on the receipt.
	points += len(receipt.Items) / 2 * 5

	// Rule 5: If the trimmed length of the item description is a multiple of 3,
	// multiply the price by 0.2 and round up to the nearest integer.
	for _, item := range receipt.Items {
		description := strings.TrimSpace(item.ShortDescription)
		if len(description)%3 == 0 {
			priceFloat, _ := strconv.ParseFloat(item.Price, 64)
			roundedPoints := int(math.Ceil(priceFloat * 0.2))
			points += roundedPoints
		}
	}

	// Rule 6: 6 points if the day in the purchase date is odd.
	purchaseDate, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
	if purchaseDate.Day()%2 == 1 {
		points += 6
	}

	// Rule 7: 10 points if the time of purchase is after 2:00pm and before 4:00pm.
	purchaseTime, _ := time.Parse("15:04", receipt.PurchaseTime)
	if purchaseTime.After(time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)) &&
		purchaseTime.Before(time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC)) {
		points += 10
	}

	return points
}

func main() {
	receiptMap = make(map[string]int)

	r := mux.NewRouter()
	r.HandleFunc("/receipts/process", ProcessReceiptHandler).Methods("POST")
	r.HandleFunc("/receipts/{id}/points", GetPointsHandler).Methods("GET")

	port := ":8080"
	fmt.Printf("Server listening on port %s...\n", port)
	log.Fatal(http.ListenAndServe(port, r))
}
