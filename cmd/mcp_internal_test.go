package cmd

import (
	"fmt"
	"testing"
)

// TestLevenshteinBasic validates distance properties including empty/identical strings.
func TestLevenshteinBasic(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"context", "context", 0},
		{"log", "lug", 1},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s_to_%s", c.a, c.b), func(t *testing.T) {
			if got := levenshtein(c.a, c.b); got != c.want {
				t.Errorf("levenshtein(%q,%q)=%d want %d", c.a, c.b, got, c.want)
			}
			if got2 := levenshtein(c.b, c.a); got2 != c.want { // symmetry
				t.Errorf("levenshtein symmetry failed: (%q,%q)=%d want %d", c.b, c.a, got2, c.want)
			}
		})
	}
}

// TestSuggestSimilar ensures ordering prefers lower distance and substring boost.
func TestSuggestSimilar(t *testing.T) {
	target := "main-latest"
	candidates := []string{"main-prod", "other", "staging-main-latest", "dev", "main-late"}
	out := suggestSimilar(target, candidates, 3)
	if len(out) == 0 {
		t.Fatalf("expected suggestions, got none")
	}
	// Expect a candidate containing the target substring or closest edit distance early.
	if out[0] != "staging-main-latest" && out[0] != "main-late" {
		// Accept both due to heuristic ordering (substring boost vs distance tie).
		// Provide diagnostic.
		t.Errorf("unexpected first suggestion: %v", out)
	}
	// Ensure max limit honored.
	if len(out) > 3 {
		t.Errorf("expected at most 3 suggestions, got %d", len(out))
	}
}

// TestSuggestSimilarSkipsIdentical verifies identical target is skipped.
func TestSuggestSimilarSkipsIdentical(t *testing.T) {
	target := "alpha"
	out := suggestSimilar(target, []string{"alpha", "alp", "alfa"}, 5)
	for _, s := range out {
		if s == target {
			t.Errorf("identical candidate %q should be skipped", target)
		}
	}
}
