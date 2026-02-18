package search

import "sort"

const rrfK = 60

// rrfScore computes Reciprocal Rank Fusion: 1/(k+rank).
func rrfScore(rank int) float64 {
	if rank < 0 {
		return 0
	}
	return 1.0 / (float64(rrfK) + float64(rank+1))
}

// MergeCandidates merges vector and FTS candidates by case_id and reranks with RRF.
// Each candidate has cosine_similarity and optionally FTSRank. We assign ranks from
// sorting by cosine and by FTS rank, then RRF score = sum(1/(k+rank)) per candidate.
// Returns slice sorted by RRF score descending.
func MergeCandidates(candidates []Candidate) []Candidate {
	if len(candidates) == 0 {
		return nil
	}

	byID := make(map[string]*Candidate)
	for i := range candidates {
		c := &candidates[i]
		byID[c.CaseID] = c
	}

	// Rank by cosine (desc)
	byCosine := make([]*Candidate, 0, len(byID))
	for _, c := range byID {
		byCosine = append(byCosine, c)
	}
	sort.Slice(byCosine, func(i, j int) bool {
		return byCosine[i].CosineSimilarity > byCosine[j].CosineSimilarity
	})
	cosineRank := make(map[string]int)
	for i, c := range byCosine {
		cosineRank[c.CaseID] = i
	}

	// Rank by FTS (desc); only those with FTSRank > 0
	var withFTS []*Candidate
	for _, c := range byID {
		if c.FTSRank > 0 {
			withFTS = append(withFTS, c)
		}
	}
	sort.Slice(withFTS, func(i, j int) bool {
		return withFTS[i].FTSRank > withFTS[j].FTSRank
	})
	ftsRank := make(map[string]int)
	for i, c := range withFTS {
		ftsRank[c.CaseID] = i
	}

	// RRF score per case
	type scored struct {
		c     *Candidate
		score float64
	}
	var scoredList []scored
	for _, c := range byID {
		s := rrfScore(cosineRank[c.CaseID])
		if r, ok := ftsRank[c.CaseID]; ok {
			s += rrfScore(r)
		}
		scoredList = append(scoredList, scored{c: c, score: s})
	}
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	out := make([]Candidate, len(scoredList))
	for i := range scoredList {
		out[i] = *scoredList[i].c
	}
	return out
}
