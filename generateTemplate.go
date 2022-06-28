package signups

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/Boostport/mjml-go"
)

// Used to create template -- runs only when tested
func generateHtml() (string, error) {
	cwd, err := os.Getwd()
	check(err)

	input, err := os.ReadFile(path.Join(cwd, "email", "generate-template", "index.mjml"))

	output, err := mjml.ToHTML(context.Background(), string(input), mjml.WithMinify(true))

	var mjmlError mjml.Error

	if errors.As(err, &mjmlError) {
		fmt.Println(mjmlError.Message)
		fmt.Println(mjmlError.Details)
	}

	f, err := os.Create(path.Join(cwd, "email", "templates", "signup_template.html"))
	check(err)

	bytes, err := f.WriteString(output)

	fmt.Printf("wrote %d bytes\n", bytes)
	return output, err
}
