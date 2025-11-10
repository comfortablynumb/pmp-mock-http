package template

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"text/template"
	"time"
)

// RequestData holds the incoming request data for template rendering
type RequestData struct {
	Method     string
	URI        string
	Path       string
	RawQuery   string
	Headers    map[string]string
	Body       string
	RemoteAddr string
}

// NewRequestData creates RequestData from an http.Request
func NewRequestData(r *http.Request, body string) *RequestData {
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	return &RequestData{
		Method:     r.Method,
		URI:        r.RequestURI,
		Path:       r.URL.Path,
		RawQuery:   r.URL.RawQuery,
		Headers:    headers,
		Body:       body,
		RemoteAddr: r.RemoteAddr,
	}
}

// Renderer handles template rendering with helper functions
type Renderer struct {
	funcMap template.FuncMap
}

// NewRenderer creates a new template renderer with helper functions
func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: template.FuncMap{
			// String generators
			"uuid":        generateUUID,
			"randomString": randomString,
			"randomInt":   randomInt,
			"randomFloat": randomFloat,
			"randomBool":  randomBool,

			// Name generators
			"firstName":  randomFirstName,
			"lastName":   randomLastName,
			"fullName":   randomFullName,
			"email":      randomEmail,
			"username":   randomUsername,

			// Address generators
			"city":       randomCity,
			"country":    randomCountry,
			"zipCode":    randomZipCode,
			"address":    randomAddress,

			// Business generators
			"company":    randomCompany,
			"jobTitle":   randomJobTitle,

			// Internet generators
			"ipAddress":  randomIPAddress,
			"domain":     randomDomain,
			"url":        randomURL,

			// Time generators
			"now":        time.Now,
			"timestamp":  func() int64 { return time.Now().Unix() },
			"date":       func() string { return time.Now().Format("2006-01-02") },
			"datetime":   func() string { return time.Now().Format(time.RFC3339) },

			// String utilities
			"upper":      strings.ToUpper,
			"lower":      strings.ToLower,

			// Number formatting
			"formatInt":  fmt.Sprintf,
		},
	}
}

// Render renders a template string with the given request data
func (r *Renderer) Render(templateStr string, data *RequestData) (string, error) {
	tmpl, err := template.New("response").Funcs(r.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Helper function implementations

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:errcheck // best effort
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func randomInt(min, max int) int {
	if min >= max {
		return min
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

func randomFloat(min, max float64) float64 {
	n, _ := rand.Int(rand.Reader, big.NewInt(1000000))
	return min + (float64(n.Int64())/1000000.0)*(max-min)
}

func randomBool() bool {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	return n.Int64() == 1
}

var firstNames = []string{
	"James", "Mary", "John", "Patricia", "Robert", "Jennifer", "Michael", "Linda",
	"William", "Barbara", "David", "Elizabeth", "Richard", "Susan", "Joseph", "Jessica",
	"Thomas", "Sarah", "Charles", "Karen", "Christopher", "Nancy", "Daniel", "Lisa",
}

var lastNames = []string{
	"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
	"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson", "Thomas",
	"Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson", "White",
}

var cities = []string{
	"New York", "Los Angeles", "Chicago", "Houston", "Phoenix", "Philadelphia", "San Antonio",
	"San Diego", "Dallas", "San Jose", "Austin", "Jacksonville", "Fort Worth", "Columbus",
	"Charlotte", "San Francisco", "Indianapolis", "Seattle", "Denver", "Washington",
}

var countries = []string{
	"USA", "Canada", "UK", "Germany", "France", "Spain", "Italy", "Australia",
	"Japan", "China", "Brazil", "Mexico", "India", "Russia", "South Korea", "Netherlands",
}

var companies = []string{
	"Tech Corp", "Global Industries", "Innovation Inc", "Digital Solutions", "Data Systems",
	"Cloud Services", "Enterprise Co", "Mega Corp", "StartUp Ltd", "Ventures Inc",
}

var jobTitles = []string{
	"Software Engineer", "Product Manager", "Data Scientist", "DevOps Engineer", "Designer",
	"Marketing Manager", "Sales Director", "HR Manager", "Financial Analyst", "CEO",
	"CTO", "VP of Engineering", "Senior Developer", "Team Lead", "Consultant",
}

func randomFirstName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(firstNames))))
	return firstNames[n.Int64()]
}

func randomLastName() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(lastNames))))
	return lastNames[n.Int64()]
}

func randomFullName() string {
	return randomFirstName() + " " + randomLastName()
}

func randomEmail() string {
	return strings.ToLower(randomFirstName()) + "." + strings.ToLower(randomLastName()) + "@example.com"
}

func randomUsername() string {
	return strings.ToLower(randomFirstName()) + randomString(4)
}

func randomCity() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(cities))))
	return cities[n.Int64()]
}

func randomCountry() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(countries))))
	return countries[n.Int64()]
}

func randomZipCode() string {
	return fmt.Sprintf("%05d", randomInt(10000, 99999))
}

func randomAddress() string {
	return fmt.Sprintf("%d %s St", randomInt(1, 9999), randomString(8))
}

func randomCompany() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(companies))))
	return companies[n.Int64()]
}

func randomJobTitle() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(jobTitles))))
	return jobTitles[n.Int64()]
}

func randomIPAddress() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		randomInt(1, 255),
		randomInt(0, 255),
		randomInt(0, 255),
		randomInt(1, 255))
}

func randomDomain() string {
	return strings.ToLower(randomString(8)) + ".com"
}

func randomURL() string {
	return "https://" + randomDomain() + "/" + strings.ToLower(randomString(8))
}
