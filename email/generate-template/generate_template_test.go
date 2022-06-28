package generate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedHtml(t *testing.T) {
	html, err := GenerateHtml()

	if err != nil || len(html) == 0 {
		t.Fatalf("Expected generateHtml() to transpile /generate-template/index.mjml to /templates/signup_template.html")
	}

	templatePath, err := filepath.Abs(filepath.Join("..", "templates", "signup_template.html"))
	if err != nil {
		t.Fatalf("Template should exist\n%v", templatePath)
	}

	f, err := os.Create(templatePath)

	f.WriteString(html)
}
