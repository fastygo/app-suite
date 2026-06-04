package storage

import (
	"context"

	"github.com/fastygo/platform/pkg/contracts"
)

type ComposedProvider struct {
	Port contracts.StoragePort
}

func NewComposedProvider(port contracts.StoragePort) ComposedProvider {
	return ComposedProvider{Port: port}
}

func (p ComposedProvider) WithinWorkspaceTx(ctx context.Context, runtime contracts.RuntimeContext, fn func(context.Context, contracts.StorageTx) error) error {
	ctx = contracts.WithRuntimeContext(ctx, runtime)
	return p.Port.WithinWorkspaceTx(ctx, runtime.WorkspaceID, fn)
}
