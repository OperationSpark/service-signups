package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/Boostport/mjml-go"
)

// Used to create template -- runs only when tested
func main() {
	templatePath, err := filepath.Abs(path.Join("generate", "template.mjml"))
	check(err, "Error finding template path")

	input, err := os.ReadFile(templatePath)
	check(err, "Error reading template file "+templatePath)

	output, err := mjml.ToHTML(context.Background(), string(input), mjml.WithMinify(true))
	var mjmlError mjml.Error
	if errors.As(err, &mjmlError) {
		fmt.Println(mjmlError.Message)
		fmt.Println(mjmlError.Details)
		check(err, "MJML transpiling error")
	}

	outputPath, err := filepath.Abs(filepath.Join("email", "templates", "signup_template.html"))
	check(err, "Error creating output path")

	f, err := os.Create(outputPath)
	check(err, "Error creating output file")

	bytes, err := f.WriteString(output)
	check(err, "Error writing file")

	fmt.Printf("\n%sSuccessfully created template (%dkb):%s\n", "\u001b[32m", bytes/1000, "\u001b[0m")
	fmt.Printf("%sFile Located => %s%s\n\n", "\u001b[32m", outputPath, "\u001b[0m")
}

func check(err error, msg string) {
	const red = "\u001b[31m"
	const reset = "\u001b[0m"
	if err != nil {
		fmt.Printf("\n%s%s%s\n", red, msg, reset)
		panic(err)
	}
}
