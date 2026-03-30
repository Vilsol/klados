package cluster

import "context"

func noopLogger() context.Context {
	return context.Background()
}
