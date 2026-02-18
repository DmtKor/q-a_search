package metrics

import "context"

// Writer writes request metrics. Errors must not affect the main response (best-effort).
type Writer interface {
	Write(ctx context.Context, record *Record) error
}
