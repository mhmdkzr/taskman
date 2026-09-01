package codebase

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLintClean(t *testing.T) {
	repo := newGoRepo(t)

	report, err := repo.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if !report.Clean() {
		t.Errorf("report not clean: %+v", report)
	}
	if report.String() != "lint passed: no issues from go vet, staticcheck, or golangci-lint" {
		t.Errorf("String() = %q", report.String())
	}
}

func TestLintReportsVetIssue(t *testing.T) {
	repo := newGoRepo(t)
	root, err := repo.Root()
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	badVet := `package main

import "fmt"

func BadPrintf() {
	fmt.Printf("%d\n", "not a number")
}
`
	if err := os.WriteFile(filepath.Join(root, "badvet.go"), []byte(badVet), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	report, err := repo.Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	if report.Clean() {
		t.Fatal("report should not be clean")
	}
	if len(report.VetIssues) == 0 {
		t.Error("want at least one go vet issue")
	}
	if !strings.Contains(report.String(), "badvet.go") {
		t.Errorf("String() = %q, want it to mention badvet.go", report.String())
	}
}
