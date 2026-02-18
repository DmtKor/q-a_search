package search

import (
	"testing"
)

func TestMergeCandidates_Empty(t *testing.T) {
	out := MergeCandidates(nil)
	if out != nil {
		t.Fatalf("expected nil, got %v", out)
	}
	out = MergeCandidates([]Candidate{})
	if out != nil {
		t.Fatalf("expected nil, got %v", out)
	}
}

func TestMergeCandidates_Single(t *testing.T) {
	in := []Candidate{
		{CaseID: "a", Title: "A", CosineSimilarity: 0.9, FTSRank: 0},
	}
	out := MergeCandidates(in)
	if len(out) != 1 {
		t.Fatalf("expected 1, got %d", len(out))
	}
	if out[0].CaseID != "a" || out[0].CosineSimilarity != 0.9 {
		t.Errorf("unexpected candidate: %+v", out[0])
	}
}

func TestMergeCandidates_RRF_PrefersBothVectorAndFTS(t *testing.T) {
	// Two candidates: one only vector, one vector + FTS. The one with both should rank higher with RRF.
	in := []Candidate{
		{CaseID: "v-only", Title: "V", CosineSimilarity: 0.95, FTSRank: 0},
		{CaseID: "both", Title: "B", CosineSimilarity: 0.8, FTSRank: 0.1},
	}
	out := MergeCandidates(in)
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	// v-only has rank 0 in cosine, both has rank 1 in cosine. So v-only RRF from cosine = 1/61, both = 1/62.
	// So v-only should still be first when we only consider cosine. But "both" has FTS rank 0 (first in FTS), so
	// both gets 1/61 + 1/61 = 2/61, v-only gets 1/61. So both should be first.
	if out[0].CaseID != "both" {
		t.Errorf("expected first to be 'both' (has vector+FTS), got %s", out[0].CaseID)
	}
	if out[1].CaseID != "v-only" {
		t.Errorf("expected second to be 'v-only', got %s", out[1].CaseID)
	}
}

func TestMergeCandidates_ConfidencePreserved(t *testing.T) {
	in := []Candidate{
		{CaseID: "a", CosineSimilarity: 0.7, FTSRank: 0.5},
		{CaseID: "b", CosineSimilarity: 0.9, FTSRank: 0},
	}
	out := MergeCandidates(in)
	for _, c := range out {
		var expected float64
		switch c.CaseID {
		case "a":
			expected = 0.7
		case "b":
			expected = 0.9
		default:
			t.Fatalf("unknown case_id %s", c.CaseID)
		}
		if c.CosineSimilarity != expected {
			t.Errorf("case_id %s: expected confidence %.2f, got %.2f", c.CaseID, expected, c.CosineSimilarity)
		}
	}
}
