package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			fmt.Printf("[Dummy Target] Received POST /login payload: %s\n", string(body))
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid credentials"}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	fmt.Println("🛡️  Dummy target server is now listening on http://localhost:8080")
	fmt.Println("Waiting for attacks...")
	
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
