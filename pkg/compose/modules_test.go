package compose

import "testing"

func TestDefaultModulesUseAppBundles(t *testing.T) {
	bundles := Bundles()
	if len(bundles) != 2 {
		t.Fatalf("bundles = %d, want 2", len(bundles))
	}
	if bundles[0].Manifest().ModuleID != "cms" {
		t.Fatalf("first bundle module = %q, want cms", bundles[0].Manifest().ModuleID)
	}
	if bundles[1].Manifest().ModuleID != "crm" {
		t.Fatalf("second bundle module = %q, want crm", bundles[1].Manifest().ModuleID)
	}
	modules := DefaultModules()
	if len(modules) != 3 {
		t.Fatalf("modules = %d, want cms + crm + monitoring", len(modules))
	}
}
