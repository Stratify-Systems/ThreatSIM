package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func main() {
	http.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		
		// User A (Admin)
		if req.Username == "admin" {
			w.Write([]byte(`{"data": {"token": "token_admin_123", "user": {"id": "100"}}}`))
			return
		}
		
		// User B (Guest)
		if req.Username == "guest" {
			w.Write([]byte(`{"data": {"token": "token_guest_456", "user": {"id": "200"}}}`))
			return
		}
		
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})

	http.HandleFunc("/api/users/100/private-data", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		
		// If User B (Guest) tries to access User A's data using their token:
		if auth == "Bearer token_guest_456" {
			// VULNERABLE! The server doesn't check if the token belongs to ID 100
			// It just sees a valid token and returns the data.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"secret_data": "Admin's highly classified info!"}`))
			return
		}
		
		w.WriteHeader(http.StatusUnauthorized)
	})

	fmt.Println("Mock API Server running on http://localhost:8080")
	fmt.Println(" - Try running the IDOR simulation against it!")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
