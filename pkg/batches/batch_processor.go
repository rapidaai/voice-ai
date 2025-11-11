package batches

import (
	"context"
)

type BatchOutput struct {
	Success bool
	Error   error
}

type BatchProcessor interface {
	Process(ctx context.Context, args map[string]string) BatchOutput
}
