package runner

import (
	"testing"
	"time"

	"github.com/jungju/ggoboong/internal/githubissue"
)

func TestFormatIssuesJSONIncludesLastCommentAndComments(t *testing.T) {
	createdAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 5, 1, 12, 5, 0, 0, time.UTC)
	comments := []githubissue.Comment{
		{
			ID:        10,
			User:      githubissue.User{Login: "alice"},
			Body:      "needs work",
			URL:       "https://example.test/comments/10",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
	}
	issues := []githubissue.Issue{
		{
			Number:         1,
			Title:          "idea",
			State:          "open",
			URL:            "https://example.test/issues/1",
			Labels:         []githubissue.Label{{Name: "idea"}, {Name: "bug"}},
			LastComment:    &comments[0],
			LoadedComments: comments,
		},
	}

	output := formatIssuesJSON(issues, true)
	if got, want := len(output), 1; got != want {
		t.Fatalf("len(output) = %d, want %d", got, want)
	}
	if got, want := output[0].LastCommenter, "alice"; got != want {
		t.Fatalf("LastCommenter = %q, want %q", got, want)
	}
	if got, want := output[0].LastCommentAt, "2026-05-01T12:00:00Z"; got != want {
		t.Fatalf("LastCommentAt = %q, want %q", got, want)
	}
	if output[0].Comments == nil {
		t.Fatal("expected comments to be included")
	}
	if got, want := len(*output[0].Comments), 1; got != want {
		t.Fatalf("len(Comments) = %d, want %d", got, want)
	}
	if got, want := output[0].Labels, []string{"bug", "idea"}; !sameStrings(got, want) {
		t.Fatalf("Labels = %v, want %v", got, want)
	}
}

func TestParseUpdatedAfter(t *testing.T) {
	for _, value := range []string{"2026-05-01", "2026-05-01T09:30:00Z"} {
		if _, err := parseUpdatedAfter(value); err != nil {
			t.Fatalf("parseUpdatedAfter(%q): %v", value, err)
		}
	}

	if _, err := parseUpdatedAfter("05/01/2026"); err == nil {
		t.Fatal("expected invalid date to fail")
	}
}

func TestValidateIssuesOptionsRequiresJSONForComments(t *testing.T) {
	err := validateIssuesOptions(IssuesOptions{
		Owner:           "owner",
		Repo:            "repo",
		IncludeComments: true,
	})
	if err == nil {
		t.Fatal("expected --include-comments without --json to fail")
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
