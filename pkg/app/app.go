package app

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fastygo/app-suite/internal/appschema"
	"github.com/fastygo/app-suite/internal/views"
	frameworkapp "github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/security"
	"github.com/fastygo/platform/pkg/contracts"
)

type Options struct {
	Addr      string
	StaticDir string
	Registry  *appschema.Registry
}

func Run() error {
	application, err := NewApp(Options{})
	if err != nil {
		return err
	}
	log.Printf("app-suite http://%s/", application.Config().AppBind)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := application.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func NewApp(options Options) (*frameworkapp.App, error) {
	registry, err := registryFromOptions(options)
	if err != nil {
		return nil, err
	}
	cfg, err := frameworkConfig(options)
	if err != nil {
		return nil, err
	}
	return frameworkapp.New(cfg).
		WithSecurity(security.LoadConfig()).
		WithHealthEndpoints(cfg.HealthLivePath, cfg.HealthReadyPath).
		WithFeature(feature{registry: registry}).
		Build(), nil
}

func NewMux(options Options) *http.ServeMux {
	registry, err := registryFromOptions(options)
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(options.StaticDir))))
	registerRoutes(mux, registry)
	return mux
}

type feature struct {
	registry *appschema.Registry
}

func (f feature) ID() string {
	return "app-suite"
}

func (f feature) NavItems() []frameworkapp.NavItem {
	return nil
}

func (f feature) Routes(mux *http.ServeMux) {
	registerRoutes(mux, f.registry)
}

func registerRoutes(mux *http.ServeMux, registry *appschema.Registry) {
	adminBase := normalizeBase(registry.Profile.AdminBase, "/go-admin")
	apiBase := normalizeBase(registry.Profile.APIBase, "/go-json")
	spacesAdminBase := normalizeBase(registry.Profile.SpacesAdminBase, joinPath(adminBase, "/spaces"))
	spacesAPIBase := normalizeBase(registry.Profile.SpacesAPIBase, joinPath(apiBase, "/spaces"))
	mux.HandleFunc("GET /{$}", renderHome(registry))
	mux.HandleFunc("GET "+routePattern(adminBase, "/"), renderAdmin(registry))
	mux.HandleFunc("GET "+routePattern(spacesAdminBase, "/"), renderWorkspaceDirectory(registry))
	mux.HandleFunc("GET "+routePattern(spacesAdminBase, "/{path...}"), renderAdmin(registry))
	mux.HandleFunc("GET "+routePattern(adminBase, "/{path...}"), renderAdmin(registry))
	mux.HandleFunc("GET "+routePattern(apiBase, "/"), renderAPIRoot(registry))
	mux.HandleFunc("GET "+routePattern(spacesAPIBase, "/{path...}"), renderSpaceAPI(registry))
	mux.HandleFunc("GET "+routePattern(apiBase, "/{path...}"), renderRootAPI(registry))
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

func registryFromOptions(options Options) (*appschema.Registry, error) {
	if options.Registry != nil {
		return options.Registry, nil
	}
	profileID := "gocms-workspaces-full"
	if value := os.Getenv("APPSUITE_PROFILE"); value != "" {
		profileID = value
	}
	p, ok := appschema.ProfileByID(profileID)
	if !ok {
		return nil, unknownProfileError(profileID)
	}
	return appschema.NewRegistry(p)
}

func frameworkConfig(options Options) (frameworkapp.Config, error) {
	cfg, err := frameworkapp.LoadConfig()
	if err != nil {
		return frameworkapp.Config{}, err
	}
	if options.Addr != "" {
		cfg.AppBind = options.Addr
	} else if addr := os.Getenv("ADDR"); addr != "" {
		cfg.AppBind = addr
	}
	if options.StaticDir != "" {
		cfg.StaticDir = options.StaticDir
	} else if os.Getenv("APP_STATIC_DIR") == "" {
		root, err := repoRoot()
		if err != nil {
			return frameworkapp.Config{}, err
		}
		cfg.StaticDir = filepath.Join(root, "web", "static")
	}
	if cfg.HealthLivePath == "" {
		cfg.HealthLivePath = "/healthz"
	}
	if cfg.HealthReadyPath == "" {
		cfg.HealthReadyPath = "/readyz"
	}
	return cfg, nil
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

func normalizeBase(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = strings.TrimRight(value, "/")
	if value == "" {
		return "/"
	}
	return value
}

func joinPath(base string, path string) string {
	base = normalizeBase(base, "/")
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return base
	}
	path = "/" + strings.Trim(path, "/")
	if base == "/" {
		return path
	}
	return strings.TrimRight(base, "/") + path
}

func routePattern(base string, suffix string) string {
	joined := joinPath(base, suffix)
	if strings.HasSuffix(suffix, "{path...}") {
		return joined
	}
	return strings.TrimRight(joined, "/") + "/{$}"
}

type unknownProfileError string

func (e unknownProfileError) Error() string {
	return "unknown AppSuite profile " + string(e)
}
