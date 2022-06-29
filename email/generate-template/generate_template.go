package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Boostport/mjml-go"
)

// Used to create template -- runs only when tested
func GenerateHtml() (string, error) {

	templatePath, err := filepath.Abs("index.mjml")
	fmt.Println("Template Path: ", templatePath)
	input, err := os.ReadFile(templatePath)

	output, err := mjml.ToHTML(context.Background(), string(input), mjml.WithMinify(true))

	var mjmlError mjml.Error

	if errors.As(err, &mjmlError) {
		fmt.Println(mjmlError.Message)
		fmt.Println(mjmlError.Details)
		return "", err
	}

	outputPath, err := filepath.Abs(filepath.Join("..", "templates", "signup_template.html"))
	fmt.Println("Output Path: ", outputPath)
	f, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}

	bytes, err := f.WriteString(output)

	fmt.Printf("wrote %d bytes\n", bytes)
	return output, err
}

func main() {
	GenerateHtml()
}
