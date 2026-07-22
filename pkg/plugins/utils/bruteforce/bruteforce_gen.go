package bruteforce

import "fmt"

// GeneratePasswords generates a list of passwords of the specified count
func GeneratePasswords(numRequests int) []string {
	var passwords []string
	
	// Base dictionary to start with
	base := []string{"123456", "password", "admin", "qwerty", "12345678", "root", "toor"}
	
	for i := 0; i < numRequests; i++ {
		if i < len(base) {
			passwords = append(passwords, base[i])
		} else {
			passwords = append(passwords, fmt.Sprintf("pass_%d_test", i))
		}
	}
	return passwords
}
