package appschema

import (
	crmapp "github.com/fastygo/app-crm/pkg/app"
	gocmsapp "github.com/fastygo/app-gocms/pkg/app"
	"github.com/fastygo/platform/pkg/contracts"
	"github.com/fastygo/platform/pkg/profile"
)

const (
	CapabilityWorkspaceRoot     contracts.CapabilityID = "workspace.root.access"
	CapabilityWorkspaceSales    contracts.CapabilityID = "workspace.sales.access"
	CapabilityWorkspaceSupport  contracts.CapabilityID = "workspace.support.access"
	CapabilityWorkspaceSEO      contracts.CapabilityID = "workspace.seo.access"
	CapabilityWorkspaceOptimize contracts.CapabilityID = "workspace.optimize.access"
	CapabilityWorkspaceIdeas    contracts.CapabilityID = "workspace.ideas.access"
)

func GoCMSAdminProfile() profile.Profile {
	return gocmsapp.DefaultProfile()
}

func CRMLeadsProfile() profile.Profile {
	return crmapp.DefaultProfile()
}

func WorkspacesFullProfile() profile.Profile {
	return profile.Profile{
		ID:              "gocms-workspaces-full",
		Title:           "GoCMS Workspaces Full",
		AdminBase:       "/go-admin",
		APIBase:         "/go-json",
		PublicBase:      "/",
		SpacesAdminBase: "/go-admin/spaces",
		SpacesAPIBase:   "/go-json/spaces",
		Workspaces: []profile.Workspace{
			rootWorkspace("root", "Content Admin", "Root GoCMS admin workspace. This is not a user task space.", "layout-dashboard", "content", "cms"),
			spaceWorkspace("sales", "Sales", "CRM leads and pipeline workspace.", "pipeline", "business", 10, CapabilityWorkspaceSales, "crm"),
			spaceWorkspace("support", "Support", "Support and tickets placeholder workspace.", "inbox", "operations", 20, CapabilityWorkspaceSupport, "crm"),
			spaceWorkspace("seo", "SEO", "SEO monitoring and content optimization workspace.", "search", "growth", 30, CapabilityWorkspaceSEO, "monitoring"),
			spaceWorkspace("optimize", "Optimize", "Performance and uptime monitoring workspace.", "activity", "operations", 40, CapabilityWorkspaceOptimize, "monitoring"),
			spaceWorkspace("ideas", "Ideas", "Ideas and notes workspace.", "lightbulb", "growth", 50, CapabilityWorkspaceIdeas, "crm"),
		},
	}
}

func HeadlessProfile() profile.Profile {
	p := GoCMSAdminProfile()
	p.ID = "headless"
	p.Title = "Headless API"
	p.Workspaces[0].Title = "Headless Content API"
	p.Workspaces[0].DefaultPath = "/go-json"
	return p
}

func LocalOfflineProfile() profile.Profile {
	p := CRMLeadsProfile()
	p.ID = "local-offline"
	p.Title = "Local Offline CRM"
	p.Workspaces[0].Description = "Local/offline CRM profile inspired by OfflineCRM datasets and snapshots."
	p.Workspaces[0].Capability = "crm.offline.sync"
	return p
}

func DemoSuiteProfile() profile.Profile {
	p := WorkspacesFullProfile()
	p.ID = "demo-suite"
	p.Title = "Demo Suite"
	return p
}

func OptionalRemoteServicesProfile() profile.Profile {
	p := WorkspacesFullProfile()
	p.ID = "optional-remote-services"
	p.Title = "Optional Remote Services"
	p.Workspaces = append(p.Workspaces, spaceWorkspace("remote-support", "Remote Support", "Optional remote support workspace served by an external module.", "radio", "operations", 60, CapabilityWorkspaceSupport, "support-remote"))
	return p
}

func ProfileByID(id string) (profile.Profile, bool) {
	switch id {
	case "gocms-admin":
		return GoCMSAdminProfile(), true
	case "crm-leads":
		return CRMLeadsProfile(), true
	case "gocms-workspaces-full", "":
		return WorkspacesFullProfile(), true
	case "headless":
		return HeadlessProfile(), true
	case "local-offline":
		return LocalOfflineProfile(), true
	case "demo-suite":
		return DemoSuiteProfile(), true
	case "optional-remote-services":
		return OptionalRemoteServicesProfile(), true
	default:
		return profile.Profile{}, false
	}
}

func rootWorkspace(id contracts.WorkspaceID, title string, description string, icon string, category string, module contracts.ModuleID) profile.Workspace {
	return profile.Workspace{
		ID:          id,
		Title:       title,
		Description: description,
		Icon:        icon,
		Category:    category,
		Order:       0,
		DefaultPath: "/go-admin",
		Capability:  CapabilityWorkspaceRoot,
		Modules:     []contracts.ModuleID{module},
		Panels:      []profile.PanelMount{{ID: string(module), BasePath: "/go-admin", Default: true}},
	}
}

func spaceWorkspace(id contracts.WorkspaceID, title string, description string, icon string, category string, order int, capability contracts.CapabilityID, module contracts.ModuleID) profile.Workspace {
	return profile.Workspace{
		ID:          id,
		Title:       title,
		Description: description,
		Icon:        icon,
		Category:    category,
		Order:       order,
		DefaultPath: "/go-admin/spaces/" + string(id),
		Capability:  capability,
		Modules:     []contracts.ModuleID{module},
		Panels:      []profile.PanelMount{{ID: string(module), BasePath: "/go-admin/spaces/" + string(id), Default: true}},
	}
}
