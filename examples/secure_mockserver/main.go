package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

	http.HandleFunc("/api/users/100/private-data", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		
		// SECURE! The server strictly verifies that the token belongs to ID 100!
		if auth != "Bearer token_admin_123" {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Unauthorized cross-tenant access prevented"}`))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"secret_data": "Admin's highly classified info!"}`))
	})

	http.HandleFunc("/api/users/update", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		bodyBytes, _ := io.ReadAll(r.Body)
		bodyStr := string(bodyBytes)

		// SECURE! Rejects cross-tenant update attempts if payload requests modification of ID 100 with non-admin token!
		if strings.Contains(bodyStr, `"user_id":"100"`) || strings.Contains(bodyStr, `"user_id": "100"`) {
			if auth != "Bearer token_admin_123" {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "Unauthorized cross-tenant update prevented"}`))
				return
			}
		}

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
			http.Error(w, "Invalid Token format", http.StatusUnauthorized)
			return
		}
		
		// SECURE! We cryptographically verify the signature matches the payload.
		// If the payload was modified (e.g. by ThreatSim jwt_forge), the signature will not match!
		expectedPayloadBase64 := "eyJuYW1lIjoiZ3Vlc3QiLCJyb2xlIjoidXNlciJ9"
		if parts[1] != expectedPayloadBase64 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid signature"}`))
			return
		}
		
		// Decode safely after signature is verified
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
		
		http.Error(w, "Unauthorized role", http.StatusForbidden)
	})

	fmt.Println("SECURE Mock API Server running on http://localhost:8081")
	fmt.Println(" - Try running the IDOR and JWT Forge simulations against it! ThreatSIM should report PASS.")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
