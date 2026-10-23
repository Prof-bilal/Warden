package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/warden-sandbox/warden/internal/policy"
)

// policyData is the template context for every fixture under
// testdata/egress/policies.
type policyData struct {
	ProbeDir string
	OutDir   string
	Allow    []string
	Timeout  int
	// ExtraRead/ExtraWrite let a fixture grant additional paths for
	// specific cases (for example a writable scratch directory).
	ExtraRead  []string
	ExtraWrite []string
}

// renderPolicy renders tmplName from the policy dir, validates it with the
// real parser, and writes it to dest. Rendering with the product parser is
// deliberate: a fixture that the product would reject must fail here.
func renderPolicy(policyDir, tmplName string, data policyData, dest string) error {
	tmplPath := filepath.Join(policyDir, tmplName+".yaml.tmpl")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("parse policy template %s: %w", tmplPath, err)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(f, data); err != nil {
		f.Close()
		return fmt.Errorf("render policy %s: %w", tmplPath, err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, err := policy.Load(dest); err != nil {
		return fmt.Errorf("rendered policy %s is invalid: %w", dest, err)
	}
	return nil
}
