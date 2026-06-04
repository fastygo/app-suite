package storage

import (
	"context"
	"testing"

	"github.com/fastygo/platform/pkg/contracts"
	"github.com/fastygo/platform/pkg/contracts/contractstest"
)

func TestComposedProviderScopesSharedStorageByWorkspace(t *testing.T) {
	provider := NewComposedProvider(contractstest.NewMemoryStorage())
	cmsRuntime := contracts.RuntimeContext{ProfileID: "suite", WorkspaceID: "content", ModuleID: "cms", PrincipalID: "editor"}
	crmRuntime := contracts.RuntimeContext{ProfileID: "suite", WorkspaceID: "sales", ModuleID: "crm", PrincipalID: "seller"}

	err := provider.WithinWorkspaceTx(context.Background(), cmsRuntime, func(ctx context.Context, tx contracts.StorageTx) error {
		runtime, ok := contracts.RuntimeContextFrom(ctx)
		if !ok {
			t.Fatalf("runtime context missing in composed transaction")
		}
		if runtime != cmsRuntime {
			t.Fatalf("runtime context = %#v, want %#v", runtime, cmsRuntime)
		}
		return tx.Put(ctx, "content", "post-1", contracts.Record{"title": "CMS post"})
	})
	if err != nil {
		t.Fatal(err)
	}

	err = provider.WithinWorkspaceTx(context.Background(), crmRuntime, func(ctx context.Context, tx contracts.StorageTx) error {
		page, err := tx.List(ctx, contracts.Query{Workspace: crmRuntime.WorkspaceID, RecordType: "content"})
		if err != nil {
			return err
		}
		if page.TotalItems != 0 {
			t.Fatalf("crm workspace sees %d cms content records, want 0", page.TotalItems)
		}
		return tx.Put(ctx, "lead", "lead-1", contracts.Record{"title": "CRM lead"})
	})
	if err != nil {
		t.Fatal(err)
	}
}
