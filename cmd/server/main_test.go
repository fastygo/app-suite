package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/fastygo/app-suite/internal/appschema"
	suiteapp "github.com/fastygo/app-suite/pkg/app"
	"github.com/fastygo/platform/pkg/profile"
)

func TestAppSuiteSmokeRoutes(t *testing.T) {
	registry, err := appschema.NewRegistry(appschema.WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	application, err := suiteapp.NewApp(suiteapp.Options{Addr: "127.0.0.1:0", StaticDir: "../../web/static", Registry: registry})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	server := httptest.NewServer(application)
	defer server.Close()
	cookie := loginCookie(t, server, "admin", "admin")

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/", want: "AppSuite"},
		{path: "/go-admin/", want: "Content Admin"},
		{path: "/go-admin/spaces/", want: "Sales"},
		{path: "/go-admin/spaces/sales", want: "CRM Leads"},
		{path: "/go-admin/spaces/sales/crm/leads", want: "Leads"},
		{path: "/go-json/", want: "assembly_path"},
		{path: "/go-json/spaces/sales", want: "sales"},
		{path: "/go-json/spaces/sales/crm/leads", want: "lead"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := get(t, server, tc.path, cookie)
			if err != nil {
				t.Fatalf("get route: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200, got %d", resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			if !strings.Contains(string(body), tc.want) {
				t.Fatalf("response for %s did not contain %q", tc.path, tc.want)
			}
		})
	}
}

func TestInvalidSpaceShortcutDoesNotResolve(t *testing.T) {
	registry, err := appschema.NewRegistry(appschema.WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	application, err := suiteapp.NewApp(suiteapp.Options{Addr: "127.0.0.1:0", StaticDir: "../../web/static", Registry: registry})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	server := httptest.NewServer(application)
	defer server.Close()
	cookie := loginCookie(t, server, "admin", "admin")

	resp, err := get(t, server, "/go-admin/sales", cookie)
	if err != nil {
		t.Fatalf("get route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected shortcut to be rejected, got %d", resp.StatusCode)
	}
}

func TestCustomSuiteSpacesBaseRoutes(t *testing.T) {
	p := appschema.WorkspacesFullProfile()
	p.ID = "custom-suite"
	p.AdminBase = "/y-admin"
	p.APIBase = "/y-json"
	p.SpacesAdminBase = "/y-admin/areas"
	p.SpacesAPIBase = "/y-json/areas"
	server := testServer(t, p)
	defer server.Close()
	cookie := loginCookie(t, server, "admin", "admin")

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/y-admin/", want: "Content Admin"},
		{path: "/y-admin/areas/", want: "Sales"},
		{path: "/y-admin/areas/sales", want: "CRM Leads"},
		{path: "/y-admin/areas/sales/crm/leads", want: "Leads"},
		{path: "/y-json/", want: "assembly_path"},
		{path: "/y-json/areas/sales", want: "sales"},
		{path: "/y-json/areas/sales/crm/leads", want: "lead"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			body := getOK(t, server, tc.path, cookie)
			if !strings.Contains(body, tc.want) {
				t.Fatalf("response for %s did not contain %q: %s", tc.path, tc.want, body)
			}
		})
	}
	resp, err := http.Get(server.URL + "/go-admin/")
	if err != nil {
		t.Fatalf("get default admin route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("custom suite must not mount default /go-admin, got %d", resp.StatusCode)
	}
}

func TestAppSuiteAPIEnvelopeAndErrors(t *testing.T) {
	server := testServer(t, appschema.WorkspacesFullProfile())
	defer server.Close()
	cookie := loginCookie(t, server, "admin", "admin")

	root := getJSONMap(t, server, "/go-json/", cookie)
	for _, key := range []string{"profile", "workspace", "api_base", "admin_base", "assembly_path", "resources"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("root discovery missing %q: %#v", key, root)
		}
	}
	if strings.Contains(getOK(t, server, "/go-json/", cookie), "/go-json/crm/v1/leads") {
		t.Fatalf("AppSuite root API must not mount standalone CRM codex routes")
	}
	sales := getJSONMap(t, server, "/go-json/spaces/sales", cookie)
	for _, key := range []string{"profile", "workspace", "api_base", "admin_base", "assembly_path", "resources"} {
		if _, ok := sales[key]; !ok {
			t.Fatalf("sales discovery missing %q: %#v", key, sales)
		}
	}
	if sales["workspace"] != "sales" || sales["api_base"] != "/go-json/spaces/sales" || sales["admin_base"] != "/go-admin/spaces/sales" {
		t.Fatalf("unexpected sales discovery bases: %#v", sales)
	}
	salesBody := getOK(t, server, "/go-json/spaces/sales", cookie)
	if !strings.Contains(salesBody, "/go-json/spaces/sales/crm/leads") {
		t.Fatalf("sales discovery must list rebased CRM leads resource: %s", salesBody)
	}
	list := getJSONMap(t, server, "/go-json/spaces/sales/crm/leads", cookie)
	for _, key := range []string{"data", "resource", "workspace", "record", "total", "cross_space", "required_cap", "renderer_view"} {
		if _, ok := list[key]; !ok {
			t.Fatalf("space list envelope missing %q: %#v", key, list)
		}
	}
	if list["workspace"] != "sales" || list["record"] != "lead" {
		t.Fatalf("unexpected sales list envelope: %#v", list)
	}
	if !strings.HasPrefix(list["required_cap"].(string), "crm.") {
		t.Fatalf("expected CRM namespaced capability: %#v", list)
	}

	resp, err := get(t, server, "/go-json/crm/v1/leads", cookie)
	if err != nil {
		t.Fatalf("get standalone CRM API route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("AppSuite must not mount standalone CRM API at root, got %d", resp.StatusCode)
	}

	resp, err = get(t, server, "/go-json/spaces/sales/unknown", cookie)
	if err != nil {
		t.Fatalf("get unknown API route: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown API route, got %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("error payload must include error key: %#v", payload)
	}
}

func TestAppSuiteWorkspaceAuthorization(t *testing.T) {
	server := testServer(t, appschema.WorkspacesFullProfile())
	defer server.Close()
	client := noRedirectClient()
	resp, err := client.Get(server.URL + "/go-json/spaces/sales/crm/leads")
	if err != nil {
		t.Fatalf("get unauth sales API: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated sales API 401, got %d", resp.StatusCode)
	}
	rootCookie := loginCookie(t, server, "root", "root")
	resp, err = get(t, server, "/go-json/spaces/sales/crm/leads", rootCookie)
	if err != nil {
		t.Fatalf("get sales as root-only: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("root-only principal must not access sales space, got %d", resp.StatusCode)
	}
	salesCookie := loginCookie(t, server, "sales", "sales")
	resp, err = get(t, server, "/go-json/", salesCookie)
	if err != nil {
		t.Fatalf("get root API as sales-only: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("sales-only principal must not access root API, got %d", resp.StatusCode)
	}
	body := getOK(t, server, "/go-json/spaces/sales/crm/leads", salesCookie)
	if !strings.Contains(body, `"workspace":"sales"`) {
		t.Fatalf("sales principal should access sales API: %s", body)
	}
}

func TestFrameworkHealthEndpoints(t *testing.T) {
	server := testServer(t, appschema.WorkspacesFullProfile())
	defer server.Close()

	for _, path := range []string{"/healthz", "/readyz"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected %s 200, got %d", path, resp.StatusCode)
		}
	}
}

func testServer(t *testing.T, p profile.Profile) *httptest.Server {
	t.Helper()
	registry, err := appschema.NewRegistry(p)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	application, err := suiteapp.NewApp(suiteapp.Options{Addr: "127.0.0.1:0", StaticDir: "../../web/static", Registry: registry})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return httptest.NewServer(application)
}

func getOK(t *testing.T, server *httptest.Server, path string, cookie *http.Cookie) string {
	t.Helper()
	resp, err := get(t, server, path, cookie)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %s 200, got %d", path, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s body: %v", path, err)
	}
	return string(body)
}

func getJSONMap(t *testing.T, server *httptest.Server, path string, cookie *http.Cookie) map[string]any {
	t.Helper()
	resp, err := get(t, server, path, cookie)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected %s 200, got %d", path, resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return payload
}

func get(t *testing.T, server *httptest.Server, path string, cookie *http.Cookie) (*http.Response, error) {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		return nil, err
	}
	if cookie != nil {
		request.AddCookie(cookie)
	}
	return noRedirectClient().Do(request)
}

func loginCookie(t *testing.T, server *httptest.Server, identifier string, password string) *http.Cookie {
	t.Helper()
	body := getOK(t, server, "/go-login", nil)
	token := hiddenValue(body, "action_token")
	form := url.Values{}
	form.Set("action_token", token)
	form.Set("identifier", identifier)
	form.Set("password", password)
	request, err := http.NewRequest(http.MethodPost, server.URL+"/go-login", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := noRedirectClient().Do(request)
	if err != nil {
		t.Fatalf("post login: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		payload, _ := io.ReadAll(response.Body)
		t.Fatalf("login returned %d: %s", response.StatusCode, payload)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == "appsuite_session" {
			return cookie
		}
	}
	t.Fatalf("login did not issue appsuite_session")
	return nil
}

func noRedirectClient() *http.Client {
	return &http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}}
}

func hiddenValue(body string, name string) string {
	needle := `name="` + name + `" value="`
	start := strings.Index(body, needle)
	if start < 0 {
		return ""
	}
	start += len(needle)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}
