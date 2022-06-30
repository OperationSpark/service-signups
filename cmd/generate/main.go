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

	var defaultGoTemplate = `package signups
type WelcomeValues struct {
	DisplayName string
	SessionDate string
	SessionTime string
}

// https://stackoverflow.com/questions/13904441/whats-the-best-way-to-bundle-static-resources-in-a-go-program

`

	templatePath, err := filepath.Abs(path.Join("template.mjml"))
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
	var goTemplateHtml = fmt.Sprintf("const InfoSessionHtml = `%s\n`", output)
	var goTemplate = fmt.Sprintf("%s\n%s", defaultGoTemplate, goTemplateHtml)

	goTemplateOutPath, err := filepath.Abs(path.Join("..", "..", "info_session_template.go"))
	check(err, "Error creating 'goTemplateOutPath'")
	goTemplateFile, err := os.Create(goTemplateOutPath)
	check(err, "Error creating 'goTemplateFile'")

	goBytes, err := goTemplateFile.WriteString(goTemplate)
	check(err, "Error: 'goTemplateFile'")

	fmt.Printf("\n%sSuccessfully created info_session_template.go (%dkb):%s\n", "\u001b[32m", goBytes/1000, "\u001b[0m")
	fmt.Printf("%sFile Located => %s%s\n\n", "\u001b[32m", goTemplateOutPath, "\u001b[0m")

}

func check(err error, msg string) {
	const red = "\u001b[31m"
	const reset = "\u001b[0m"
	if err != nil {
		fmt.Printf("\n%s%s%s\n", red, msg, reset)
		panic(err)
	}
}
