package connectors

import "context"

// Connector ideated from python-service-template
// An interface which provide common behavior for all the data source connectors.
type Connector interface {
	Connect(ctx context.Context) error
	Name() string
	IsConnected(ctx context.Context) bool
	Disconnect(ctx context.Context) error
}
