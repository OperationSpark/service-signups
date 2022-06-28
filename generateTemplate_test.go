package signups

import (
	"bytes"
	"os"
	"path"
	"testing"
	"time"
)

func TestGeneratedHtml(t *testing.T) {
	var html, err = generateHtml()

	if err != nil || len(html) == 0 {
		t.Fatalf("Expected generateHtml() to transpile /generate-template/index.mjml to /templates/signup_template.html")
	}
	cwd, err := os.Getwd()
	check(err)

	f, err := os.Create(path.Join(cwd, "email", "templates", "signup_template.html"))
	check(err)

	f.WriteString(html)
}
func TestRenderHtml(t *testing.T) {
	sessionStartDate, _ := time.Parse(time.RFC822, "02 Feb 22 15:00 UTC")
	test := Signup{
		NameFirst:        "Peter",
		NameLast:         "Barnum",
		StartDateTime:    sessionStartDate,
		Email:            "henri@email.com",
		Cell:             "555-123-4567",
		Referrer:         "Word of mouth",
		ReferrerResponse: "Jane Smith",
		Cohort:           "is-feb-28-22-12pm",
	}

	var b bytes.Buffer
	err := test.html(&b)
	check(err)
	cwd, err := os.Getwd()
	check(err)

	f, err := os.Create(path.Join(cwd, "rendered.ignore.html"))
	check(err)

	f.WriteString(b.String())

}
