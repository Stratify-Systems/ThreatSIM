package bruteforce

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// GeneratePasswords generates a list of passwords of specified count.
// If wordlistPath is provided, passwords are read line-by-line from the file.
func GeneratePasswords(numRequests int, wordlistPath string) ([]string, error) {
	var base []string

	if wordlistPath != "" {
		file, err := os.Open(wordlistPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open wordlist file %q: %w", wordlistPath, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				base = append(base, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed reading wordlist file %q: %w", wordlistPath, err)
		}

		if len(base) == 0 {
			return nil, fmt.Errorf("wordlist file %q is empty", wordlistPath)
		}
	} else {
		// Default built-in password dictionary
		base = []string{"123456", "password", "admin", "qwerty", "12345678", "root", "toor"}
	}

	var passwords []string
	for i := 0; i < numRequests; i++ {
		if i < len(base) {
			passwords = append(passwords, base[i])
		} else {
			passwords = append(passwords, fmt.Sprintf("pass_%d_test", i))
		}
	}

	return passwords, nil
}
