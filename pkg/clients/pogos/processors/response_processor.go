package clients_response_processors

import (
	"context"

	clients_pogos "github.com/lexatic/web-backend/pkg/clients/pogos"
)

type ResponseProcessor[T string | []*clients_pogos.Interaction] interface {
	Process(ctx context.Context, request *clients_pogos.RequestData[T]) *clients_pogos.PromptResponse
}
