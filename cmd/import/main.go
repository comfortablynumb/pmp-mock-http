package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/openapi"
)

func main() {
	// Define flags
	input := flag.String("input", "", "Path or URL to OpenAPI/Swagger spec (required)")
	output := flag.String("output", "mocks/imported.yaml", "Output path for generated mocks")
	generateExamples := flag.Bool("generate-examples", false, "Generate example responses from schemas")
	flag.Parse()

	// Validate input
	if *input == "" {
		fmt.Println("Error: --input flag is required")
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("PMP Mock HTTP - OpenAPI/Swagger Importer\n")
	log.Printf("==========================================\n")

	// Create parser
	parser := openapi.NewParser(*generateExamples)

	// Parse spec
	var mockSpec *models.MockSpec
	var err error

	if isURL(*input) {
		log.Printf("Fetching spec from URL: %s\n", *input)
		mockSpec, err = parser.ParseURL(*input)
	} else {
		log.Printf("Reading spec from file: %s\n", *input)
		mockSpec, err = parser.ParseFile(*input)
	}

	if err != nil {
		log.Fatalf("Failed to parse spec: %v\n", err)
	}

	log.Printf("Generated %d mocks\n", len(mockSpec.Mocks))

	// Save mocks
	if err := openapi.SaveMocks(mockSpec, *output); err != nil {
		log.Fatalf("Failed to save mocks: %v\n", err)
	}

	log.Printf("✓ Successfully imported OpenAPI spec\n")
	log.Printf("✓ Mocks saved to: %s\n", *output)
	log.Printf("\nTo use these mocks, start the server with:\n")
	log.Printf("  ./pmp-mock-http --mocks-dir %s\n", *output)
}

// isURL checks if the input is a URL
func isURL(input string) bool {
	return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
}
