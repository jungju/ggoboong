package githubissue

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListIssuesLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "2" {
			t.Fatalf("per_page = %q, want %q", got, "2")
		}
		fmt.Fprint(w, `[
			{"number":1,"title":"first","state":"open","html_url":"https://example.test/1","labels":[]},
			{"number":2,"title":"second","state":"open","html_url":"https://example.test/2","labels":[]}
		]`)
	}))
	defer server.Close()

	client := NewClient(server.Client(), "token")
	client.baseURL = server.URL

	issues, err := client.ListIssues(context.Background(), "owner", "repo", ListIssuesOptions{Limit: 2})
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}

	if got, want := len(issues), 2; got != want {
		t.Fatalf("len(issues) = %d, want %d", got, want)
	}
}

func TestListIssuesFiltersLastCommenter(t *testing.T) {
	commentRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/issues":
			fmt.Fprint(w, `[
				{"number":1,"title":"bot answered","state":"open","html_url":"https://example.test/1","labels":[],"comments":1},
				{"number":2,"title":"needs reply","state":"open","html_url":"https://example.test/2","labels":[],"comments":1},
				{"number":3,"title":"no comments","state":"open","html_url":"https://example.test/3","labels":[],"comments":0}
			]`)
		case "/repos/owner/repo/issues/1/comments":
			commentRequests++
			fmt.Fprint(w, `[{"id":101,"user":{"login":"GGOBOONG"},"body":"done","html_url":"https://example.test/c1"}]`)
		case "/repos/owner/repo/issues/2/comments":
			commentRequests++
			fmt.Fprint(w, `[{"id":201,"user":{"login":"alice"},"body":"ping","html_url":"https://example.test/c2"}]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), "token")
	client.baseURL = server.URL

	issues, err := client.ListIssues(context.Background(), "owner", "repo", ListIssuesOptions{LastCommenterNot: "ggoboong"})
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}

	if got, want := issueNumbers(issues), []int{2, 3}; !sameInts(got, want) {
		t.Fatalf("issue numbers = %v, want %v", got, want)
	}
	if got, want := commentRequests, 2; got != want {
		t.Fatalf("comment requests = %d, want %d", got, want)
	}
}

func TestListIssuesLimitAppliesAfterLastCommenterFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/issues":
			switch r.URL.Query().Get("page") {
			case "1":
				fmt.Fprint(w, `[{"number":1,"title":"bot answered","state":"open","html_url":"https://example.test/1","labels":[],"comments":1}]`)
			case "2":
				fmt.Fprint(w, `[{"number":2,"title":"needs reply","state":"open","html_url":"https://example.test/2","labels":[],"comments":1}]`)
			default:
				fmt.Fprint(w, `[]`)
			}
		case "/repos/owner/repo/issues/1/comments":
			fmt.Fprint(w, `[{"id":101,"user":{"login":"ggoboong"},"body":"done","html_url":"https://example.test/c1"}]`)
		case "/repos/owner/repo/issues/2/comments":
			fmt.Fprint(w, `[{"id":201,"user":{"login":"alice"},"body":"ping","html_url":"https://example.test/c2"}]`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.Client(), "token")
	client.baseURL = server.URL

	issues, err := client.ListIssues(context.Background(), "owner", "repo", ListIssuesOptions{
		Limit:            1,
		LastCommenterNot: "ggoboong",
	})
	if err != nil {
		t.Fatalf("list issues: %v", err)
	}

	if got, want := issueNumbers(issues), []int{2}; !sameInts(got, want) {
		t.Fatalf("issue numbers = %v, want %v", got, want)
	}
}

func TestHasAllLabels(t *testing.T) {
	labels := []Label{{Name: "bug"}, {Name: "help wanted"}}

	if !hasAllLabels(labels, []string{"bug", "help wanted"}) {
		t.Fatal("expected issue to include all requested labels")
	}
	if !hasAllLabels(labels, []string{"BUG"}) {
		t.Fatal("expected label matching to be case-insensitive")
	}
	if hasAllLabels(labels, []string{"bug", "done"}) {
		t.Fatal("expected missing label to fail")
	}
}

func TestHasAnyLabel(t *testing.T) {
	labels := []Label{{Name: "bug"}, {Name: "help wanted"}}

	if !hasAnyLabel(labels, []string{"done", "BUG"}) {
		t.Fatal("expected matching label to be found")
	}
	if hasAnyLabel(labels, []string{"done"}) {
		t.Fatal("expected unrelated label not to match")
	}
}

func issueNumbers(issues []Issue) []int {
	numbers := make([]int, 0, len(issues))
	for _, issue := range issues {
		numbers = append(numbers, issue.Number)
	}
	return numbers
}

func sameInts(a, b []int) bool {
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
