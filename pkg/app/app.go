package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fastygo/app-suite/internal/appschema"
	"github.com/fastygo/app-suite/internal/policy"
	"github.com/fastygo/app-suite/internal/views"
	frameworkapp "github.com/fastygo/framework/pkg/app"
	frameworkauth "github.com/fastygo/framework/pkg/auth"
	"github.com/fastygo/framework/pkg/web/security"
	"github.com/fastygo/platform/pkg/contracts"
)

type Options struct {
	Addr       string
	StaticDir  string
	Registry   *appschema.Registry
	SessionKey string
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
	authBoundary := authFromOptions(options, cfg, registry)
	return frameworkapp.New(cfg).
		WithSecurity(security.LoadConfig()).
		WithHealthEndpoints(cfg.HealthLivePath, cfg.HealthReadyPath).
		WithFeature(feature{registry: registry, auth: authBoundary}).
		Build(), nil
}

func NewMux(options Options) *http.ServeMux {
	registry, err := registryFromOptions(options)
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(options.StaticDir))))
	cfg, err := frameworkConfig(options)
	if err != nil {
		panic(err)
	}
	registerRoutes(mux, registry, authFromOptions(options, cfg, registry))
	return mux
}

type feature struct {
	registry *appschema.Registry
	auth     authBoundary
}

func (f feature) ID() string {
	return "app-suite"
}

func (f feature) NavItems() []frameworkapp.NavItem {
	return nil
}

func (f feature) Routes(mux *http.ServeMux) {
	registerRoutes(mux, f.registry, f.auth)
}

func registerRoutes(mux *http.ServeMux, registry *appschema.Registry, authBoundary authBoundary) {
	adminBase := normalizeBase(registry.Profile.AdminBase, "/go-admin")
	apiBase := normalizeBase(registry.Profile.APIBase, "/go-json")
	spacesAdminBase := normalizeBase(registry.Profile.SpacesAdminBase, joinPath(adminBase, "/spaces"))
	spacesAPIBase := normalizeBase(registry.Profile.SpacesAPIBase, joinPath(apiBase, "/spaces"))
	mux.HandleFunc("GET /{$}", renderHome(registry))
	mux.HandleFunc("GET /go-login", authBoundary.renderLogin(registry))
	mux.HandleFunc("POST /go-login", authBoundary.completeLogin(registry))
	mux.HandleFunc("GET /go-logout", authBoundary.completeLogout)
	mux.HandleFunc("POST /go-logout", authBoundary.completeLogout)
	mux.HandleFunc("GET "+routePattern(adminBase, "/"), authBoundary.protectAdmin(registry, renderAdmin(registry)))
	mux.HandleFunc("GET "+routePattern(spacesAdminBase, "/"), authBoundary.protectWorkspaceDirectory(registry, renderWorkspaceDirectory(registry)))
	mux.HandleFunc("GET "+routePattern(spacesAdminBase, "/{path...}"), authBoundary.protectAdmin(registry, renderAdmin(registry)))
	mux.HandleFunc("GET "+routePattern(adminBase, "/{path...}"), authBoundary.protectAdmin(registry, renderAdmin(registry)))
	mux.HandleFunc("GET "+routePattern(apiBase, "/"), authBoundary.protectAPI(registry, renderAPIRoot(registry)))
	mux.HandleFunc("GET "+routePattern(spacesAPIBase, "/{path...}"), authBoundary.protectAPI(registry, renderSpaceAPI(registry)))
	mux.HandleFunc("GET "+routePattern(apiBase, "/{path...}"), authBoundary.protectAPI(registry, renderRootAPI(registry)))
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

type authBoundary struct {
	session  frameworkauth.CookieSession[contracts.SessionClaims]
	secret   string
	grants   policy.MemoryEvaluator
	password map[string]string
}

type actionToken struct {
	Action string `json:"action"`
	Exp    int64  `json:"exp"`
}

func authFromOptions(options Options, cfg frameworkapp.Config, registry *appschema.Registry) authBoundary {
	secret := firstNonEmpty(options.SessionKey, cfg.SessionKey, "appsuite-development-session-secret-32-bytes")
	return authBoundary{
		secret: secret,
		session: frameworkauth.CookieSession[contracts.SessionClaims]{
			Name:     "appsuite_session",
			Path:     "/",
			Secret:   secret,
			TTL:      8 * time.Hour,
			SameSite: http.SameSiteLaxMode,
			HTTPOnly: true,
		},
		password: map[string]string{"admin": "admin", "root": "root", "sales": "sales"},
		grants:   policy.NewMemoryEvaluator(defaultGrants(registry)),
	}
}

func defaultGrants(registry *appschema.Registry) map[contracts.PrincipalID]policy.PrincipalGrants {
	admin := policy.PrincipalGrants{}
	rootOnly := policy.PrincipalGrants{}
	salesOnly := policy.PrincipalGrants{}
	for id, workspace := range registry.Workspaces {
		caps := []contracts.CapabilityID{workspace.Workspace.Capability, "content.read", "content.write", "admin.access", "crm.access", "crm.lead.read", "crm.lead.write"}
		admin[id] = policy.CapabilitySet(caps...)
		if workspace.IsRoot {
			rootOnly[id] = policy.CapabilitySet(workspace.Workspace.Capability, "content.read", "content.write", "admin.access")
		}
		if id == "sales" {
			salesOnly[id] = policy.CapabilitySet(workspace.Workspace.Capability, "crm.access", "crm.lead.read", "crm.lead.write")
		}
	}
	return map[contracts.PrincipalID]policy.PrincipalGrants{
		"admin": admin,
		"root":  rootOnly,
		"sales": salesOnly,
	}
}

func (a authBoundary) renderLogin(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		token, err := a.actionToken("login", 10*time.Minute)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprintf(w, `<!doctype html><html><body><main><h1>AppSuite Login</h1><form method="post" action="/go-login"><input type="hidden" name="action_token" value="%s"><input type="hidden" name="next" value="%s/"><label>Username <input name="identifier"></label><label>Password <input name="password" type="password"></label><button type="submit">Sign in</button></form></main></body></html>`, token, normalizeBase(registry.Profile.AdminBase, "/go-admin"))
	}
}

func (a authBoundary) completeLogin(registry *appschema.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if !a.validActionToken(r.FormValue("action_token"), "login") {
			http.Error(w, "invalid action token", http.StatusForbidden)
			return
		}
		identifier := strings.TrimSpace(r.FormValue("identifier"))
		if a.password[identifier] == "" || a.password[identifier] != r.FormValue("password") {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		if err := a.session.Issue(w, contracts.SessionClaims{PrincipalID: identifier, ProfileID: registry.Profile.ID}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		next := firstNonEmpty(r.FormValue("next"), normalizeBase(registry.Profile.AdminBase, "/go-admin")+"/")
		http.Redirect(w, r, next, http.StatusSeeOther)
	}
}

func (a authBoundary) completeLogout(w http.ResponseWriter, r *http.Request) {
	a.session.Clear(w)
	http.Redirect(w, r, "/go-login", http.StatusSeeOther)
}

func (a authBoundary) protectWorkspaceDirectory(registry *appschema.Registry, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.allow(w, r, registry, registry.WorkspaceRoot, "") {
			return
		}
		next(w, r)
	}
}

func (a authBoundary) protectAdmin(registry *appschema.Registry, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID, _, ok := registry.ResolveAdmin(strings.TrimRight(r.URL.Path, "/"))
		if !ok {
			next(w, r)
			return
		}
		capability := contracts.CapabilityID("")
		if binding, ok := registry.Screens[strings.TrimRight(r.URL.Path, "/")]; ok {
			capability = binding.Screen.Capability
		}
		if !a.allow(w, r, registry, workspaceID, capability) {
			return
		}
		next(w, r)
	}
}

func (a authBoundary) protectAPI(registry *appschema.Registry, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workspaceID, _, ok := registry.ResolveAPI(strings.TrimRight(r.URL.Path, "/"))
		if !ok {
			next(w, r)
			return
		}
		capability := contracts.CapabilityID("")
		if binding, ok := registry.APIResource(r.URL.Path); ok {
			capability = binding.Screen.Capability
		}
		if !a.allowJSON(w, r, registry, workspaceID, capability) {
			return
		}
		next(w, r)
	}
}

func (a authBoundary) allow(w http.ResponseWriter, r *http.Request, registry *appschema.Registry, workspaceID contracts.WorkspaceID, capability contracts.CapabilityID) bool {
	if claims, ok := a.session.Read(r); ok {
		if a.allowed(r, registry, claims, workspaceID, capability) {
			return true
		}
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	http.Redirect(w, r, "/go-login", http.StatusSeeOther)
	return false
}

func (a authBoundary) allowJSON(w http.ResponseWriter, r *http.Request, registry *appschema.Registry, workspaceID contracts.WorkspaceID, capability contracts.CapabilityID) bool {
	claims, ok := a.session.Read(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, apiError("authorization required"))
		return false
	}
	if !a.allowed(r, registry, claims, workspaceID, capability) {
		writeJSON(w, http.StatusForbidden, apiError("missing capability"))
		return false
	}
	return true
}

func (a authBoundary) allowed(r *http.Request, registry *appschema.Registry, claims contracts.SessionClaims, workspaceID contracts.WorkspaceID, capability contracts.CapabilityID) bool {
	workspace, ok := registry.Workspaces[workspaceID]
	if !ok {
		return false
	}
	runtime := contracts.RuntimeContext{ProfileID: claims.ProfileID, WorkspaceID: workspaceID, ModuleID: firstModule(workspace.Workspace.Modules), PrincipalID: contracts.PrincipalID(claims.PrincipalID)}
	workspaceDecision, err := a.grants.Evaluate(r.Context(), runtime.PolicyRequest("workspace", contracts.PolicyRead, workspace.Workspace.Capability))
	if err != nil || !workspaceDecision.Allowed {
		return false
	}
	if capability == "" {
		return true
	}
	resourceDecision, err := a.grants.Evaluate(r.Context(), runtime.PolicyRequest("resource", contracts.PolicyRead, capability))
	return err == nil && resourceDecision.Allowed
}

func (a authBoundary) actionToken(action string, ttl time.Duration) (string, error) {
	return frameworkauth.SignedEncode(actionToken{Action: action, Exp: time.Now().Add(ttl).Unix()}, a.secret)
}

func (a authBoundary) validActionToken(raw string, action string) bool {
	var token actionToken
	if err := frameworkauth.SignedDecode(raw, a.secret, &token); err != nil {
		return false
	}
	return token.Action == action && token.Exp >= time.Now().Unix()
}

func firstModule(modules []contracts.ModuleID) contracts.ModuleID {
	if len(modules) == 0 {
		return ""
	}
	return modules[0]
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type unknownProfileError string

func (e unknownProfileError) Error() string {
	return "unknown AppSuite profile " + string(e)
}
