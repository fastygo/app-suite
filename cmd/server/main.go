package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fastygo/app-suite/internal/appschema"
	"github.com/fastygo/app-suite/internal/views"
	"github.com/fastygo/platform/pkg/contracts"
)

func main() {
	addr := envOr("ADDR", "127.0.0.1:8080")
	profileID := envOr("APPSUITE_PROFILE", "gocms-workspaces-full")
	p, ok := appschema.ProfileByID(profileID)
	if !ok {
		log.Fatalf("unknown AppSuite profile %q", profileID)
	}
	root, err := repoRoot()
	if err != nil {
		log.Fatal(err)
	}
	registry, err := appschema.NewRegistry(p)
	if err != nil {
		log.Fatal(err)
	}
	mux := newMux(filepath.Join(root, "web", "static"), registry)
	log.Printf("app-suite preview http://%s/ profile=%s", addr, p.ID)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func newMux(staticDir string, registry *appschema.Registry) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.HandleFunc("GET /{$}", renderHome(registry))
	mux.HandleFunc("GET /go-admin/{$}", renderAdmin(registry))
	mux.HandleFunc("GET /go-admin/spaces/{$}", renderWorkspaceDirectory(registry))
	mux.HandleFunc("GET /go-admin/spaces/{path...}", renderAdmin(registry))
	mux.HandleFunc("GET /go-admin/{path...}", renderAdmin(registry))
	mux.HandleFunc("GET /go-json/{$}", renderAPIRoot(registry))
	mux.HandleFunc("GET /go-json/spaces/{path...}", renderSpaceAPI(registry))
	mux.HandleFunc("GET /go-json/{path...}", renderRootAPI(registry))
	return mux
}

func renderHome(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.Home(registry.Profile.Title).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func renderWorkspaceDirectory(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.WorkspaceDirectory(registry.Directory(), registry.Switcher(registry.WorkspaceRoot)).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func renderAdmin(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimRight(r.URL.Path, "/")
		workspaceID, _, ok := registry.ResolveAdmin(path)
		if !ok {
			writeJSON(w, http.StatusNotFound, apiError("admin screen not found"))
			return
		}
		screen, err := registry.Screen(path)
		if err != nil {
			writeJSON(w, http.StatusNotFound, apiError("admin screen not found"))
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := views.Page(screen.Title, screen, registry.Switcher(workspaceID)).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func renderAPIRoot(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, apiDiscovery(registry, registry.WorkspaceRoot))
	}
}

func renderRootAPI(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if binding, ok := registry.APIResource(r.URL.Path); ok {
			writeJSON(w, http.StatusOK, apiList(binding))
			return
		}
		writeJSON(w, http.StatusNotFound, apiError("route not found"))
	}
}

func renderSpaceAPI(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID, base, ok := registry.ResolveAPI(r.URL.Path)
		if !ok {
			writeJSON(w, http.StatusNotFound, apiError("space route not found"))
			return
		}
		if strings.TrimRight(r.URL.Path, "/") == base {
			writeJSON(w, http.StatusOK, apiDiscovery(registry, workspaceID))
			return
		}
		if binding, ok := registry.APIResource(r.URL.Path); ok {
			writeJSON(w, http.StatusOK, apiList(binding))
			return
		}
		writeJSON(w, http.StatusNotFound, apiError("space route not found"))
	}
}

func apiDiscovery(registry *appschema.Registry, workspaceID contracts.WorkspaceID) map[string]any {
	workspace, ok := registry.Workspaces[workspaceID]
	if !ok {
		return map[string]any{"error": "workspace not found"}
	}
	resources := []string{}
	for path, binding := range registry.APIScreens {
		if binding.WorkspaceID == workspaceID {
			resources = append(resources, path)
		}
	}
	return map[string]any{
		"profile":       registry.Profile.ID,
		"workspace":     workspace.Workspace.ID,
		"title":         workspace.Workspace.Title,
		"api_base":      workspace.APIBase,
		"admin_base":    workspace.AdminBase,
		"assembly_path": registry.AssemblyPath,
		"resources":     resources,
	}
}

func apiList(binding appschema.ScreenBinding) map[string]any {
	return map[string]any{
		"data":          []map[string]any{},
		"resource":      binding.Resource.ID,
		"workspace":     binding.WorkspaceID,
		"record":        binding.Record.ID,
		"total":         0,
		"cross_space":   "explicit-policy-required",
		"required_cap":  binding.Screen.Capability,
		"renderer_view": binding.Screen.View,
	}
}

func apiError(message string) map[string]string {
	return map[string]string{"error": message}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
