package search

import (
	"context"
	"errors"
	"testing"

	"github.com/yourusername/project/internal/auth"
)

type mockEmbedding struct {
	vec []float32
	err error
}

func (m *mockEmbedding) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.vec != nil {
		return m.vec, nil
	}
	return []float32{0.1}, nil
}

type mockRepo struct {
	candidates []Candidate
	err        error
}

func (m *mockRepo) SearchApproved(ctx context.Context, params SearchParams) ([]Candidate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.candidates, nil
}

type mockRenderer struct {
	out string
	err error
}

func (m *mockRenderer) Render(ctx context.Context, template string, userContext map[string]interface{}) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if m.out != "" {
		return m.out, nil
	}
	return template, nil
}

type mockTickets struct {
	id  string
	url string
	err error
}

func (m *mockTickets) CreateLowConfidenceTicket(ctx context.Context, data LowConfidenceTicketData) (string, string, error) {
	if m.err != nil {
		return "", "", m.err
	}
	return m.id, m.url, nil
}

type mockSettings struct {
	threshold   float64
	defaultTopK int
	err         error
}

func (m *mockSettings) GetEffectiveSearchSettings(ctx context.Context, principal *auth.Principal) (float64, int, error) {
	if m.err != nil {
		return 0, 0, m.err
	}
	return m.threshold, m.defaultTopK, nil
}

func TestService_Search_NoCategory_SearchesAll(t *testing.T) {
	svc := &Service{
		Embedding:   &mockEmbedding{vec: []float32{0.1}},
		Repo:        &mockRepo{candidates: []Candidate{{CaseID: "c1", Title: "T", ResponseTemplate: "Hi", CosineSimilarity: 0.9}}},
		Renderer:    &mockRenderer{},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{threshold: 0.7, defaultTopK: 10},
	}
	req := SearchRequest{Query: "test"}
	resp, err := svc.Search(context.Background(), req, &auth.Principal{TokenType: "app"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(resp.Chunks))
	}
	if resp.Chunks[0].CaseID != "c1" || resp.Chunks[0].Confidence != 0.9 {
		t.Errorf("chunk: %+v", resp.Chunks[0])
	}
	if resp.Ticket != nil {
		t.Error("expected no ticket when confidence >= threshold")
	}
}

func TestService_Search_TopKApplied(t *testing.T) {
	var many []Candidate
	for i := 0; i < 20; i++ {
		id := "c" + string(rune('a'+i))
		many = append(many, Candidate{CaseID: id, Title: "T", ResponseTemplate: "X", CosineSimilarity: 0.9 - float64(i)*0.01})
	}
	svc := &Service{
		Embedding:   &mockEmbedding{vec: []float32{0.1}},
		Repo:        &mockRepo{candidates: many},
		Renderer:    &mockRenderer{},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{threshold: 0.7, defaultTopK: 5},
	}
	req := SearchRequest{Query: "test"}
	resp, err := svc.Search(context.Background(), req, &auth.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Chunks) != 5 {
		t.Errorf("expected 5 chunks (defaultTopK), got %d", len(resp.Chunks))
	}
}

func TestService_Search_TicketOnlyOnLowConfidence(t *testing.T) {
	highConf := []Candidate{
		{CaseID: "c1", Title: "T", ResponseTemplate: "X", CosineSimilarity: 0.9},
	}
	lowConf := []Candidate{
		{CaseID: "c1", Title: "T", ResponseTemplate: "X", CosineSimilarity: 0.5},
	}
	tickets := &mockTickets{id: "tid", url: "/tickets/tid"}

	svcHigh := &Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{candidates: highConf},
		Renderer:    &mockRenderer{},
		Tickets:     tickets,
		AppSettings: &mockSettings{threshold: 0.7, defaultTopK: 10},
	}
	respHigh, err := svcHigh.Search(context.Background(), SearchRequest{Query: "q"}, &auth.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if respHigh.Ticket != nil {
		t.Error("expected no ticket when confidence 0.9 >= 0.7")
	}

	svcLow := &Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{candidates: lowConf},
		Renderer:    &mockRenderer{},
		Tickets:     tickets,
		AppSettings: &mockSettings{threshold: 0.7, defaultTopK: 10},
	}
	respLow, err := svcLow.Search(context.Background(), SearchRequest{Query: "q"}, &auth.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if respLow.Ticket == nil {
		t.Fatal("expected ticket when confidence 0.5 < 0.7")
	}
	if respLow.Ticket.ID != "tid" || respLow.Ticket.URL != "/tickets/tid" {
		t.Errorf("ticket: %+v", respLow.Ticket)
	}
}

func TestService_Search_ConfidenceIsCosine(t *testing.T) {
	candidates := []Candidate{
		{CaseID: "c1", Title: "T1", ResponseTemplate: "R1", CosineSimilarity: 0.88},
	}
	svc := &Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{candidates: candidates},
		Renderer:    &mockRenderer{out: "rendered"},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{threshold: 0.9, defaultTopK: 10},
	}
	resp, err := svc.Search(context.Background(), SearchRequest{Query: "q"}, &auth.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatal("expected 1 chunk")
	}
	if resp.Chunks[0].Confidence != 0.88 {
		t.Errorf("confidence must be cosine (0.88), got %.2f", resp.Chunks[0].Confidence)
	}
	if resp.Chunks[0].Text != "rendered" {
		t.Errorf("text must be rendered template, got %q", resp.Chunks[0].Text)
	}
}

func TestService_Search_TopKValidation(t *testing.T) {
	svc := &Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{},
		Renderer:    &mockRenderer{},
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{threshold: 0.7, defaultTopK: 10},
	}
	zero := 0
	fiftyOne := 51
	for _, req := range []SearchRequest{
		{Query: "q", TopK: &zero},
		{Query: "q", TopK: &fiftyOne},
	} {
		_, err := svc.Search(context.Background(), req, &auth.Principal{})
		if !errors.Is(err, ErrInvalidTopK) {
			t.Errorf("expected ErrInvalidTopK for top_k=%v, got %v", req.TopK, err)
		}
	}
}

func TestService_Search_RendererMissingFields(t *testing.T) {
	// TemplateRenderer contract: render even with missing user_context fields
	svc := &Service{
		Embedding:   &mockEmbedding{},
		Repo:        &mockRepo{candidates: []Candidate{{CaseID: "c1", Title: "T", ResponseTemplate: "Hello {{.name}}", CosineSimilarity: 0.8}}},
		Renderer:    &mockRenderer{out: "Hello"}, // simulates empty/missing name
		Tickets:     &mockTickets{},
		AppSettings: &mockSettings{threshold: 0.7, defaultTopK: 10},
	}
	resp, err := svc.Search(context.Background(), SearchRequest{Query: "q", UserContext: map[string]interface{}{}}, &auth.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatal("expected 1 chunk")
	}
	// We accept whatever Render returns (or template on error)
	if resp.Chunks[0].Text != "Hello" {
		t.Errorf("expected rendered text, got %q", resp.Chunks[0].Text)
	}
}
