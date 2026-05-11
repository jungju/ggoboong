package githubissue

import "testing"

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
