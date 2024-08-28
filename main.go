package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

// Review represents a review submitted by a user
type Review struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Review string `json:"review"`
}

// Slice to store reviews
var reviews []Review

// Mutex to synchronize access to the reviews slice
var mutex = &sync.Mutex{}

// Counter to generate unique IDs for reviews
var idCounter = 0

// File to persist reviews
const reviewsFile = "reviews.json"

func main() {
	// Load existing reviews from the file
	loadReviews()

	http.HandleFunc("/reviews", reviewsHandler)
	http.HandleFunc("/delete-review", deleteReviewHandler) // New handler for deleting a review

	fmt.Println("Server is listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// loadReviews loads reviews from the file at startup
func loadReviews() {
	file, err := ioutil.ReadFile(reviewsFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, no reviews to load
			reviews = []Review{}
			return
		}
		log.Fatalf("Failed to load reviews: %v", err)
	}

	// Parse JSON data into the reviews slice
	err = json.Unmarshal(file, &reviews)
	if err != nil {
		log.Fatalf("Failed to parse reviews: %v", err)
	}

	// Set the idCounter to the highest ID found
	for _, review := range reviews {
		if review.ID > idCounter {
			idCounter = review.ID
		}
	}
}

// saveReviews saves the current reviews slice to a file
func saveReviews() {
	data, err := json.MarshalIndent(reviews, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal reviews: %v", err)
		return
	}

	err = ioutil.WriteFile(reviewsFile, data, 0644)
	if err != nil {
		log.Printf("Failed to write reviews to file: %v", err)
	}
}

// reviewsHandler handles both POST and GET requests for reviews
func reviewsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		handlePostReview(w, r)
	case http.MethodGet:
		handleGetReviews(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePostReview handles the submission of a new review
func handlePostReview(w http.ResponseWriter, r *http.Request) {
	// Parse the JSON request body
	var newReview Review
	if err := json.NewDecoder(r.Body).Decode(&newReview); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Lock the mutex before modifying the slice
	mutex.Lock()
	defer mutex.Unlock()

	// Assign a unique ID to the new review
	idCounter++
	newReview.ID = idCounter

	reviews = append(reviews, newReview)

	// Save reviews to the file
	saveReviews()

	// Respond with success and the assigned ID
	response := map[string]interface{}{"success": true, "id": newReview.ID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetReviews handles fetching all submitted reviews
func handleGetReviews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Lock the mutex before reading the slice
	mutex.Lock()
	defer mutex.Unlock()

	json.NewEncoder(w).Encode(reviews)
}

// deleteReviewHandler handles the deletion of a review by ID
func deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the JSON request body to get the ID of the review to delete
	var requestData struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Lock the mutex before modifying the slice
	mutex.Lock()
	defer mutex.Unlock()

	// Find and remove the review with the specified ID
	index := -1
	for i, review := range reviews {
		if review.ID == requestData.ID {
			index = i
			break
		}
	}

	if index == -1 {
		http.Error(w, "Review not found", http.StatusNotFound)
		return
	}

	// Remove the review from the slice
	reviews = append(reviews[:index], reviews[index+1:]...)

	// Save reviews to the file
	saveReviews()

	// Respond with success
	response := map[string]bool{"success": true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
