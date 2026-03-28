package search

// SearchParams are inputs for retrieval (approved-only, optional category).
type SearchParams struct {
	QueryVector []float32
	QueryFTS    string
	Category    string // empty = all categories
	Limit       int    // e.g. 2*top_k for each branch
}

// Candidate is a single result from search (vector and/or FTS) before merge.
type Candidate struct {
	CaseID          string
	Title           string
	ResponseTemplate string
	Category        string
	CosineSimilarity float64
	FTSRank         float64 // 0 if from vector-only path
}

// SearchRequest is the API request body (OpenAPI SearchRequest).
type SearchRequest struct {
	Query                    string                 `json:"query"`
	Category                 string                 `json:"category,omitempty"`
	TopK                     *int                   `json:"top_k,omitempty"`
	UserContext              map[string]interface{} `json:"user_context,omitempty"`
	NoTicketOnLowConfidence  bool                  `json:"no_ticket_on_low_confidence,omitempty"`
}

// Chunk is one item in the response (OpenAPI Chunk).
type Chunk struct {
	CaseID     string  `json:"case_id"`
	Title      string  `json:"title"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// TicketRef is the optional ticket in response (OpenAPI TicketRef).
type TicketRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// SearchResponse is the API response (OpenAPI SearchResponse).
type SearchResponse struct {
	Chunks []Chunk   `json:"chunks"`
	Ticket *TicketRef `json:"ticket,omitempty"`
}
