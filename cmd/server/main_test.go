package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/", want: "AppSuite"},
		{path: "/go-admin/", want: "Content Admin"},
		{path: "/go-admin/spaces", want: "Sales"},
		{path: "/go-admin/spaces/sales", want: "CRM Leads"},
		{path: "/go-admin/spaces/sales/crm/leads", want: "Leads"},
		{path: "/go-json/", want: "assembly_path"},
		{path: "/go-json/spaces/sales", want: "sales"},
		{path: "/go-json/spaces/sales/crm/leads", want: "lead"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(server.URL + tc.path)
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

	resp, err := http.Get(server.URL + "/go-admin/sales")
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

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/y-admin/", want: "Content Admin"},
		{path: "/y-admin/areas", want: "Sales"},
		{path: "/y-admin/areas/sales", want: "CRM Leads"},
		{path: "/y-admin/areas/sales/crm/leads", want: "Leads"},
		{path: "/y-json/", want: "assembly_path"},
		{path: "/y-json/areas/sales", want: "sales"},
		{path: "/y-json/areas/sales/crm/leads", want: "lead"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			body := getOK(t, server, tc.path)
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

	root := getJSONMap(t, server, "/go-json/")
	for _, key := range []string{"profile", "workspace", "api_base", "admin_base", "assembly_path", "resources"} {
		if _, ok := root[key]; !ok {
			t.Fatalf("root discovery missing %q: %#v", key, root)
		}
	}
	list := getJSONMap(t, server, "/go-json/spaces/sales/crm/leads")
	for _, key := range []string{"data", "resource", "workspace", "record", "total", "cross_space", "required_cap", "renderer_view"} {
		if _, ok := list[key]; !ok {
			t.Fatalf("space list envelope missing %q: %#v", key, list)
		}
	}
	if list["workspace"] != "sales" || list["record"] != "lead" {
		t.Fatalf("unexpected sales list envelope: %#v", list)
	}

	resp, err := http.Get(server.URL + "/go-json/spaces/sales/unknown")
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

func getOK(t *testing.T, server *httptest.Server, path string) string {
	t.Helper()
	resp, err := http.Get(server.URL + path)
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

func getJSONMap(t *testing.T, server *httptest.Server, path string) map[string]any {
	t.Helper()
	resp, err := http.Get(server.URL + path)
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
