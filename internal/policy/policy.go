package policy

import (
	"context"

	"github.com/fastygo/platform/pkg/contracts"
)

type PrincipalGrants map[contracts.WorkspaceID]map[contracts.CapabilityID]struct{}

type MemoryEvaluator struct {
	Grants map[contracts.PrincipalID]PrincipalGrants
}

func NewMemoryEvaluator(grants map[contracts.PrincipalID]PrincipalGrants) MemoryEvaluator {
	return MemoryEvaluator{Grants: grants}
}

func (e MemoryEvaluator) Evaluate(_ context.Context, request contracts.PolicyRequest) (contracts.PolicyDecision, error) {
	principalGrants, ok := e.Grants[request.Principal]
	if !ok {
		return contracts.PolicyDecision{Allowed: false, Reason: "unknown principal"}, nil
	}
	workspaceGrants, ok := principalGrants[request.Workspace]
	if !ok {
		return contracts.PolicyDecision{Allowed: false, Reason: "workspace denied"}, nil
	}
	if request.Capability == "" {
		return contracts.PolicyDecision{Allowed: true, Reason: "allowed"}, nil
	}
	if _, ok := workspaceGrants[request.Capability]; !ok {
		return contracts.PolicyDecision{Allowed: false, Reason: "missing capability"}, nil
	}
	return contracts.PolicyDecision{Allowed: true, Reason: "allowed"}, nil
}

func CapabilitySet(capabilities ...contracts.CapabilityID) map[contracts.CapabilityID]struct{} {
	set := map[contracts.CapabilityID]struct{}{}
	for _, capability := range capabilities {
		set[capability] = struct{}{}
	}
	return set
}
