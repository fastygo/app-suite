package appschema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/platform/pkg/profile"
	"github.com/fastygo/platform/pkg/toolset"
)

func TestProductProfilesValidateRootContracts(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    profile.Profile
	}{
		{name: "gocms admin", p: GoCMSAdminProfile()},
		{name: "crm leads", p: CRMLeadsProfile()},
		{name: "workspaces full", p: WorkspacesFullProfile()},
		{name: "headless", p: HeadlessProfile()},
		{name: "local offline", p: LocalOfflineProfile()},
		{name: "demo suite", p: DemoSuiteProfile()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := (&tc.p).Validate(); err != nil {
				t.Fatalf("validate profile: %v", err)
			}
			if tc.p.AdminBase != "/go-admin" || tc.p.APIBase != "/go-json" {
				t.Fatalf("profile must preserve root contracts, got %q %q", tc.p.AdminBase, tc.p.APIBase)
			}
		})
	}
}

func TestProfileJSONFilesLoad(t *testing.T) {
	for _, name := range []string{
		"gocms-admin.json",
		"crm-leads.json",
		"gocms-workspaces-full.json",
		"headless.json",
		"local-offline.json",
		"demo-suite.json",
	} {
		t.Run(name, func(t *testing.T) {
			file, err := os.Open(filepath.Join("..", "..", "profiles", name))
			if err != nil {
				t.Fatalf("open profile json: %v", err)
			}
			defer file.Close()
			if _, err := profile.LoadJSON(file); err != nil {
				t.Fatalf("load profile json: %v", err)
			}
		})
	}
}

func TestWorkspacesFullUsesSpacesOverlaySemantics(t *testing.T) {
	registry, err := NewRegistry(WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	if registry.RootIsTaskSpace() {
		t.Fatalf("root admin must not be treated as a task space")
	}
	if len(registry.Directory().Items) != 5 {
		t.Fatalf("expected 5 task spaces, got %d", len(registry.Directory().Items))
	}
	if workspace, base, ok := registry.ResolveAdmin("/go-admin/spaces/sales"); !ok || workspace != "sales" || base != "/go-admin/spaces/sales" {
		t.Fatalf("sales admin overlay did not resolve: workspace=%q base=%q ok=%v", workspace, base, ok)
	}
	if workspace, base, ok := registry.ResolveAPI("/go-json/spaces/sales"); !ok || workspace != "sales" || base != "/go-json/spaces/sales" {
		t.Fatalf("sales API overlay did not resolve: workspace=%q base=%q ok=%v", workspace, base, ok)
	}
	if workspace, _, ok := registry.ResolveAdmin("/go-admin"); !ok || workspace != "root" {
		t.Fatalf("root admin did not resolve to root workspace")
	}
	if _, err := registry.Screen("/go-admin/sales"); err == nil {
		t.Fatalf("spaces must not mount directly below /go-admin/{space}")
	}
}

func TestSingleAndMultiWorkspaceProfilesUseSameAssemblyPath(t *testing.T) {
	single, err := NewRegistry(CRMLeadsProfile())
	if err != nil {
		t.Fatalf("new single registry: %v", err)
	}
	multi, err := NewRegistry(WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new multi registry: %v", err)
	}
	if single.AssemblyPath != "modulehost.Assemble(profile)" || multi.AssemblyPath != single.AssemblyPath {
		t.Fatalf("profiles must use one assembly path, got %q and %q", single.AssemblyPath, multi.AssemblyPath)
	}
	if len(single.Assemblies) != 1 {
		t.Fatalf("single profile should assemble one workspace")
	}
	if len(multi.Assemblies) < 2 {
		t.Fatalf("multi profile should assemble multiple workspaces")
	}
}

func TestSalesWorkspaceResolvesCRMResources(t *testing.T) {
	registry, err := NewRegistry(WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	screen, err := registry.Screen("/go-admin/spaces/sales/crm/leads")
	if err != nil {
		t.Fatalf("resolve sales leads screen: %v", err)
	}
	if screen.Record != "lead" {
		t.Fatalf("expected CRM lead screen, got record %q", screen.Record)
	}
	if binding, ok := registry.APIResource("/go-json/spaces/sales/crm/leads"); !ok || binding.WorkspaceID != "sales" {
		t.Fatalf("expected sales CRM API resource")
	}
}

func TestWorkspaceAccessAndCrossWorkspacePolicyMetadata(t *testing.T) {
	registry, err := NewRegistry(WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	sales := registry.Workspaces["sales"].Workspace
	if sales.Capability != CapabilityWorkspaceSales {
		t.Fatalf("expected sales capability %q, got %q", CapabilityWorkspaceSales, sales.Capability)
	}
	policies := registry.CrossWorkspacePolicySamples()
	if len(policies) != 2 {
		t.Fatalf("expected policy samples")
	}
	if policies[0].CrossWorkspaceMode != toolset.CrossWorkspaceSameProfile {
		t.Fatalf("expected same-profile policy")
	}
	if policies[1].CrossWorkspaceMode != toolset.CrossWorkspaceRequiresCapability {
		t.Fatalf("expected requires-capability policy")
	}
}
