package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCommentBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comment.md")
	if err := os.WriteFile(path, []byte("hello\n"), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	body, err := readCommentBody(path)
	if err != nil {
		t.Fatalf("readCommentBody: %v", err)
	}
	if got, want := body, "hello\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestReadCommentBodyRejectsEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comment.md")
	if err := os.WriteFile(path, []byte(" \n"), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	_, err := readCommentBody(path)
	if err == nil {
		t.Fatal("expected empty body to fail")
	}
	if !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("error = %q, want empty body message", err)
	}
}

func TestValidateCommentOptions(t *testing.T) {
	err := validateCommentOptions(CommentOptions{
		Owner:    "owner",
		Repo:     "repo",
		Issue:    1,
		BodyFile: "comment.md",
	})
	if err != nil {
		t.Fatalf("validateCommentOptions: %v", err)
	}

	err = validateCommentOptions(CommentOptions{Owner: "owner", Repo: "repo", Issue: 1})
	if err == nil {
		t.Fatal("expected missing --body-file to fail")
	}
}
