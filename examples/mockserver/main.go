package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
			// A valid 3-part JWT format for {"name":"guest", "role":"user"}
			w.Write([]byte(`{"data": {"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiZ3Vlc3QiLCJyb2xlIjoidXNlciJ9.signature123", "user": {"id": "200"}}}`))
			return
		}
		
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// VULNERABLE! Server never rate-limits or locks out after repeated attempts
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid credentials"}`))
	})

	http.HandleFunc("/api/users/100/private-data", func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			// VULNERABLE! Reflects any untrusted origin and sets Access-Control-Allow-Credentials: true
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

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

	http.HandleFunc("/api/users/update", func(w http.ResponseWriter, r *http.Request) {
		// VULNERABLE! Server accepts update request for any user ID without checking tenant ownership!
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "updated"}`))
	})

	http.HandleFunc("/api/admin/secrets", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		
		token := strings.TrimPrefix(auth, "Bearer ")
		parts := strings.Split(token, ".")
		if len(parts) != 3 {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}
		
		// VULNERABLE! We decode the payload and trust it blindly without checking the signature!
		payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			http.Error(w, "Decode error", http.StatusUnauthorized)
			return
		}
		
		var payload map[string]interface{}
		json.Unmarshal(payloadBytes, &payload)
		
		if payload["role"] == "admin" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"secrets": "JWT Forgery bypass successful!"}`))
			return
		}
		
		http.Error(w, "invalid signature", http.StatusUnauthorized)
	})

	fmt.Println("Mock API Server running on http://localhost:8080")
	fmt.Println(" - Try running the IDOR and JWT Forge simulations against it!")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
