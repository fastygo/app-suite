package appschema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fastygo/app-suite/pkg/compose"
	"github.com/fastygo/platform/pkg/contracts"
	"github.com/fastygo/platform/pkg/modulehost"
	"github.com/fastygo/platform/pkg/panel"
	"github.com/fastygo/platform/pkg/profile"
	"github.com/fastygo/platform/pkg/render"
	"github.com/fastygo/platform/pkg/toolset"
)

type Registry struct {
	Profile       profile.Profile
	Assemblies    []modulehost.WorkspaceAssembly
	Workspaces    map[contracts.WorkspaceID]WorkspaceRuntime
	Screens       map[string]ScreenBinding
	APIScreens    map[string]ScreenBinding
	AssemblyPath  string
	WorkspaceRoot contracts.WorkspaceID
}

type WorkspaceRuntime struct {
	Workspace profile.Workspace
	AdminBase string
	APIBase   string
	IsRoot    bool
	Assembly  modulehost.WorkspaceAssembly
}

type ScreenBinding struct {
	WorkspaceID contracts.WorkspaceID
	Resource    panel.Resource[contracts.CapabilityID]
	Record      toolset.RecordTypeDefinition
	Screen      render.ScreenModel
}

type WorkspaceDirectoryModel struct {
	Title string
	Items []WorkspaceItem
}

type WorkspaceItem struct {
	ID          contracts.WorkspaceID
	Title       string
	Description string
	Icon        string
	Order       int
	Category    string
	DefaultPath string
	Capability  contracts.CapabilityID
}

type WorkspaceSwitcherModel struct {
	CurrentWorkspaceID contracts.WorkspaceID
	RootAdminLink      string
	CurrentTitle       string
	CurrentIcon        string
	CurrentCategory    string
	Available          []WorkspaceItem
}

func NewDefaultRegistry() (*Registry, error) {
	return NewRegistry(WorkspacesFullProfile())
}

func NewRegistry(p profile.Profile) (*Registry, error) {
	return NewRegistryWithModules(p, defaultModules()...)
}

func NewRegistryWithModules(p profile.Profile, modules ...contracts.Module) (*Registry, error) {
	host, err := modulehost.New(modules...)
	if err != nil {
		return nil, err
	}
	assemblies, err := host.Assemble(p)
	if err != nil {
		return nil, err
	}
	if len(assemblies) == 0 {
		return nil, fmt.Errorf("profile %q assembled without workspaces", p.ID)
	}
	registry := &Registry{
		Profile:       p,
		Assemblies:    assemblies,
		Workspaces:    map[contracts.WorkspaceID]WorkspaceRuntime{},
		Screens:       map[string]ScreenBinding{},
		APIScreens:    map[string]ScreenBinding{},
		AssemblyPath:  "modulehost.Assemble(profile)",
		WorkspaceRoot: assemblies[0].Workspace.ID,
	}
	if registry.Profile.SpacesAdminBase == "" {
		registry.Profile.SpacesAdminBase = strings.TrimRight(registry.Profile.AdminBase, "/") + "/spaces"
	}
	if registry.Profile.SpacesAPIBase == "" {
		registry.Profile.SpacesAPIBase = strings.TrimRight(registry.Profile.APIBase, "/") + "/spaces"
	}
	for i, assembly := range assemblies {
		isRoot := i == 0
		adminBase := strings.TrimRight(registry.Profile.AdminBase, "/")
		apiBase := strings.TrimRight(registry.Profile.APIBase, "/")
		if !isRoot {
			adminBase = strings.TrimRight(registry.Profile.SpacesAdminBase, "/") + "/" + string(assembly.Workspace.ID)
			apiBase = strings.TrimRight(registry.Profile.SpacesAPIBase, "/") + "/" + string(assembly.Workspace.ID)
		}
		runtime := WorkspaceRuntime{Workspace: assembly.Workspace, AdminBase: adminBase, APIBase: apiBase, IsRoot: isRoot, Assembly: assembly}
		registry.Workspaces[assembly.Workspace.ID] = runtime
		registry.indexWorkspace(runtime)
	}
	return registry, nil
}

func defaultModules() []contracts.Module {
	return compose.DefaultModules()
}

func (r *Registry) ResolveAdmin(path string) (contracts.WorkspaceID, string, bool) {
	clean := strings.TrimRight(path, "/")
	if clean == "" {
		clean = "/"
	}
	for _, workspace := range r.sortedWorkspacesByAdminBase() {
		if clean == workspace.AdminBase || strings.HasPrefix(clean, workspace.AdminBase+"/") {
			return workspace.Workspace.ID, workspace.AdminBase, true
		}
	}
	return "", "", false
}

func (r *Registry) ResolveAPI(path string) (contracts.WorkspaceID, string, bool) {
	clean := strings.TrimRight(path, "/")
	if clean == "" {
		clean = "/"
	}
	for _, workspace := range r.sortedWorkspacesByAPIBase() {
		if clean == workspace.APIBase || strings.HasPrefix(clean, workspace.APIBase+"/") {
			return workspace.Workspace.ID, workspace.APIBase, true
		}
	}
	return "", "", false
}

func (r *Registry) DashboardScreen(workspaceID contracts.WorkspaceID) (render.ScreenModel, error) {
	workspace, ok := r.Workspaces[workspaceID]
	if !ok {
		return render.ScreenModel{}, fmt.Errorf("unknown workspace %q", workspaceID)
	}
	screen := workspaceDashboard(workspace.Workspace)
	screen.Metadata = copyMetadata(screen.Metadata)
	screen.Metadata["workspace_id"] = string(workspace.Workspace.ID)
	screen.Metadata["admin_base"] = workspace.AdminBase
	screen.Metadata["api_base"] = workspace.APIBase
	return screen, nil
}

func (r *Registry) Screen(path string) (render.ScreenModel, error) {
	clean := strings.TrimRight(path, "/")
	if binding, ok := r.Screens[clean]; ok {
		return binding.Screen, nil
	}
	workspaceID, base, ok := r.ResolveAdmin(clean)
	if ok && clean == base {
		return r.DashboardScreen(workspaceID)
	}
	return render.ScreenModel{}, fmt.Errorf("unknown admin screen path %q", path)
}

func (r *Registry) Directory() WorkspaceDirectoryModel {
	items := r.spaceItems()
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Category == items[j].Category {
			if items[i].Order == items[j].Order {
				return items[i].Title < items[j].Title
			}
			return items[i].Order < items[j].Order
		}
		return items[i].Category < items[j].Category
	})
	return WorkspaceDirectoryModel{Title: "Workspaces", Items: items}
}

func (r *Registry) Switcher(current contracts.WorkspaceID) WorkspaceSwitcherModel {
	model := WorkspaceSwitcherModel{
		CurrentWorkspaceID: current,
		RootAdminLink:      strings.TrimRight(r.Profile.AdminBase, "/") + "/",
		Available:          r.spaceItems(),
	}
	if workspace, ok := r.Workspaces[current]; ok {
		model.CurrentTitle = workspace.Workspace.Title
		model.CurrentIcon = workspace.Workspace.Icon
		model.CurrentCategory = workspace.Workspace.Category
	}
	return model
}

func (r *Registry) APIResource(path string) (ScreenBinding, bool) {
	binding, ok := r.APIScreens[strings.TrimRight(path, "/")]
	return binding, ok
}

func (r *Registry) RootIsTaskSpace() bool {
	return false
}

func (r *Registry) CrossWorkspacePolicySamples() []toolset.RelationPolicy {
	return []toolset.RelationPolicy{
		{CrossWorkspaceMode: toolset.CrossWorkspaceSameProfile, ReadOnly: true},
		{CrossWorkspaceMode: toolset.CrossWorkspaceRequiresCapability, Capability: toolset.CapabilityID(CapabilityWorkspaceSales)},
	}
}

func (r *Registry) indexWorkspace(workspace WorkspaceRuntime) {
	for _, resource := range workspace.Assembly.Context.Resources {
		record := recordFor(resource, workspace.Assembly.Context.Records)
		adminPath := workspacePath(workspace.AdminBase, resource.BasePath)
		screen := render.ResourceTableScreen(resource, record)
		screen.Metadata = copyMetadata(screen.Metadata)
		screen.Metadata["workspace_id"] = string(workspace.Workspace.ID)
		screen.Metadata["admin_base"] = workspace.AdminBase
		screen.Metadata["api_base"] = workspace.APIBase
		r.Screens[adminPath] = ScreenBinding{WorkspaceID: workspace.Workspace.ID, Resource: resource, Record: record, Screen: screen}
		r.APIScreens[workspacePath(workspace.APIBase, resource.BasePath)] = r.Screens[adminPath]
	}
	for _, page := range workspace.Assembly.Context.Pages {
		resource := resourceForPage(page, workspace.Assembly.Context.Resources)
		record := recordFor(resource, workspace.Assembly.Context.Records)
		adminPath := workspacePath(workspace.AdminBase, page.Path)
		screen := screenForPage(page, resource, record)
		screen.Metadata = copyMetadata(screen.Metadata)
		screen.Metadata["workspace_id"] = string(workspace.Workspace.ID)
		screen.Metadata["admin_base"] = workspace.AdminBase
		screen.Metadata["api_base"] = workspace.APIBase
		r.Screens[adminPath] = ScreenBinding{WorkspaceID: workspace.Workspace.ID, Resource: resource, Record: record, Screen: screen}
		r.APIScreens[workspacePath(workspace.APIBase, page.Path)] = r.Screens[adminPath]
	}
}

func (r *Registry) spaceItems() []WorkspaceItem {
	items := []WorkspaceItem{}
	for _, workspace := range r.Workspaces {
		if workspace.IsRoot {
			continue
		}
		items = append(items, WorkspaceItem{
			ID:          workspace.Workspace.ID,
			Title:       workspace.Workspace.Title,
			Description: workspace.Workspace.Description,
			Icon:        workspace.Workspace.Icon,
			Order:       workspace.Workspace.Order,
			Category:    workspace.Workspace.Category,
			DefaultPath: workspace.AdminBase,
			Capability:  workspace.Workspace.Capability,
		})
	}
	return items
}

func (r *Registry) sortedWorkspacesByAdminBase() []WorkspaceRuntime {
	workspaces := make([]WorkspaceRuntime, 0, len(r.Workspaces))
	for _, workspace := range r.Workspaces {
		workspaces = append(workspaces, workspace)
	}
	sort.SliceStable(workspaces, func(i, j int) bool {
		return len(workspaces[i].AdminBase) > len(workspaces[j].AdminBase)
	})
	return workspaces
}

func (r *Registry) sortedWorkspacesByAPIBase() []WorkspaceRuntime {
	workspaces := make([]WorkspaceRuntime, 0, len(r.Workspaces))
	for _, workspace := range r.Workspaces {
		workspaces = append(workspaces, workspace)
	}
	sort.SliceStable(workspaces, func(i, j int) bool {
		return len(workspaces[i].APIBase) > len(workspaces[j].APIBase)
	})
	return workspaces
}

func workspacePath(base string, resourceBase string) string {
	base = strings.TrimRight(base, "/")
	suffix := strings.TrimPrefix(strings.TrimRight(resourceBase, "/"), "/go-admin")
	if suffix == "" {
		return base
	}
	if !strings.HasPrefix(suffix, "/") {
		suffix = "/" + suffix
	}
	return base + suffix
}

func recordFor(resource panel.Resource[contracts.CapabilityID], records []toolset.RecordTypeDefinition) toolset.RecordTypeDefinition {
	for _, record := range records {
		if string(record.ID) == string(resource.ID) || strings.TrimSuffix(string(resource.ID), "s") == string(record.ID) {
			return record
		}
	}
	if len(records) > 0 {
		return records[0]
	}
	return toolset.RecordTypeDefinition{ID: toolset.RecordTypeID(resource.ID), Label: resource.Label}
}

func resourceForPage(page panel.Page[contracts.CapabilityID], resources []panel.Resource[contracts.CapabilityID]) panel.Resource[contracts.CapabilityID] {
	for _, resource := range resources {
		if strings.HasPrefix(strings.TrimRight(page.Path, "/"), strings.TrimRight(resource.BasePath, "/")) {
			return resource
		}
	}
	if len(resources) > 0 {
		return resources[0]
	}
	return panel.Resource[contracts.CapabilityID]{ID: panel.ResourceID(page.ID), Label: page.Title, BasePath: page.Path, Table: page.Table}
}

func screenForPage(page panel.Page[contracts.CapabilityID], resource panel.Resource[contracts.CapabilityID], record toolset.RecordTypeDefinition) render.ScreenModel {
	screen := render.ScreenModel{
		ID:         string(page.ID),
		Title:      page.Title,
		View:       render.ViewTable,
		Record:     record.ID,
		Resource:   resource.ID,
		Columns:    page.Table.Columns,
		Fields:     page.Form.Fields,
		Capability: page.Capability,
		Metadata:   map[string]string{},
	}
	if screen.Capability == "" {
		screen.Capability = capabilityFor(resource, panel.OperationList)
	}
	switch page.ID {
	case "lead-kanban":
		screen.View = render.ViewKanban
		screen.Fallback = render.ViewTable
		screen.Metadata["group_field"] = "stage"
	case "lead-detail":
		screen.View = render.ViewType("detail")
	}
	return screen
}

func workspaceDashboard(workspace profile.Workspace) render.ScreenModel {
	title := workspace.Title
	if len(workspace.Modules) > 0 {
		switch workspace.Modules[0] {
		case "cms":
			title = "GoCMS Admin"
		case "crm":
			title = "CRM Leads"
		}
	}
	return render.ScreenModel{
		ID:       string(workspace.ID) + "-dashboard",
		Title:    title,
		View:     render.ViewType("dashboard"),
		Resource: "dashboard",
		Metadata: map[string]string{
			"workspace": string(workspace.ID),
		},
	}
}

func capabilityFor(resource panel.Resource[contracts.CapabilityID], operation panel.ResourceOperation) contracts.CapabilityID {
	for _, capability := range resource.Capabilities {
		if capability.Operation == operation {
			return capability.Capability
		}
	}
	return ""
}

func copyMetadata(metadata map[string]string) map[string]string {
	copy := map[string]string{}
	for key, value := range metadata {
		copy[key] = value
	}
	return copy
}
