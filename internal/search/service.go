package search

import (
	"context"
	"math"

	"github.com/yourusername/project/internal/auth"
)

func sanitizeConfidence(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// Service runs search flow: embed -> retrieve -> merge -> top_k -> ticket if low confidence -> render.
type Service struct {
	Embedding   EmbeddingProvider
	Repo        SearchRepository
	Renderer    TemplateRenderer
	Tickets     TicketsWriter
	AppSettings AppSettingsReader
}

// MinTopK / MaxTopK are used only to validate request top_k (OpenAPI 1..50).
// Default threshold and defaultTopK come from AppSettingsReader (module 05).
const (
	MinTopK = 1
	MaxTopK = 50
)

// Search executes the full search pipeline and returns the API response.
func (s *Service) Search(ctx context.Context, req SearchRequest, principal *auth.Principal) (*SearchResponse, error) {
	threshold, defaultTopK, err := s.AppSettings.GetEffectiveSearchSettings(ctx, principal)
	if err != nil {
		return nil, err
	}
	topK := defaultTopK
	if req.TopK != nil {
		topK = *req.TopK
	}
	if topK < MinTopK || topK > MaxTopK {
		return nil, ErrInvalidTopK
	}

	queryVec, err := s.Embedding.EmbedQuery(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	params := SearchParams{
		QueryVector: queryVec,
		QueryFTS:    req.Query,
		Category:    req.Category,
		Limit:       topK * 2,
	}
	candidates, err := s.Repo.SearchApproved(ctx, params)
	if err != nil {
		return nil, err
	}

	merged := MergeCandidates(candidates)
	if len(merged) > topK {
		merged = merged[:topK]
	}

	userContext := req.UserContext
	if userContext == nil {
		userContext = make(map[string]interface{})
	}

	chunks := make([]Chunk, 0, len(merged))
	for i := range merged {
		c := &merged[i]
		text, err := s.Renderer.Render(ctx, c.ResponseTemplate, userContext)
		if err != nil {
			text = c.ResponseTemplate
		}
		chunks = append(chunks, Chunk{
			CaseID:     c.CaseID,
			Title:      c.Title,
			Text:       text,
			Confidence: sanitizeConfidence(c.CosineSimilarity),
		})
	}

	resp := &SearchResponse{Chunks: chunks}

	if len(merged) > 0 {
		top1Confidence := sanitizeConfidence(merged[0].CosineSimilarity)
		if top1Confidence < threshold {
			ticketID, ticketURL, err := s.Tickets.CreateLowConfidenceTicket(ctx, LowConfidenceTicketData{
				Query:      req.Query,
				Category:   req.Category,
				Confidence: top1Confidence,
			})
			if err != nil {
				return nil, err
			}
			resp.Ticket = &TicketRef{ID: ticketID, URL: ticketURL}
		}
	}

	return resp, nil
}
