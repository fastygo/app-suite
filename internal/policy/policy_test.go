package policy

import (
	"context"
	"testing"

	"github.com/fastygo/platform/pkg/contracts"
)

func TestMemoryEvaluatorAllowsSamePrincipalPerWorkspace(t *testing.T) {
	evaluator := NewMemoryEvaluator(map[contracts.PrincipalID]PrincipalGrants{
		"user-1": {
			"content": CapabilitySet("cms.content.read"),
			"sales":   CapabilitySet("crm.lead.read"),
		},
	})

	cmsRuntime := contracts.RuntimeContext{WorkspaceID: "content", ModuleID: "cms", PrincipalID: "user-1"}
	crmRuntime := contracts.RuntimeContext{WorkspaceID: "sales", ModuleID: "crm", PrincipalID: "user-1"}

	allowed, err := evaluator.Evaluate(context.Background(), cmsRuntime.PolicyRequest("posts", contracts.PolicyRead, "cms.content.read"))
	if err != nil {
		t.Fatal(err)
	}
	if !allowed.Allowed {
		t.Fatalf("cms read should be allowed: %+v", allowed)
	}
	allowed, err = evaluator.Evaluate(context.Background(), crmRuntime.PolicyRequest("leads", contracts.PolicyRead, "crm.lead.read"))
	if err != nil {
		t.Fatal(err)
	}
	if !allowed.Allowed {
		t.Fatalf("crm lead read should be allowed: %+v", allowed)
	}
}

func TestMemoryEvaluatorDeniesCapabilityInWrongWorkspace(t *testing.T) {
	evaluator := NewMemoryEvaluator(map[contracts.PrincipalID]PrincipalGrants{
		"user-1": {
			"content": CapabilitySet("cms.content.read"),
			"sales":   CapabilitySet("crm.lead.read"),
		},
	})

	runtime := contracts.RuntimeContext{WorkspaceID: "sales", ModuleID: "crm", PrincipalID: "user-1"}
	decision, err := evaluator.Evaluate(context.Background(), runtime.PolicyRequest("leads", contracts.PolicyUpdate, "crm.lead.write"))
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed {
		t.Fatalf("crm write should be denied without workspace capability")
	}
}

func TestMemoryEvaluatorDeniesMissingWorkspace(t *testing.T) {
	evaluator := NewMemoryEvaluator(map[contracts.PrincipalID]PrincipalGrants{
		"user-1": {
			"content": CapabilitySet("cms.content.read"),
		},
	})
	runtime := contracts.RuntimeContext{WorkspaceID: "sales", ModuleID: "crm", PrincipalID: "user-1"}
	decision, err := evaluator.Evaluate(context.Background(), runtime.PolicyRequest("leads", contracts.PolicyRead, "crm.lead.read"))
	if err != nil {
		t.Fatal(err)
	}
	if decision.Allowed {
		t.Fatalf("sales workspace should be denied")
	}
}
