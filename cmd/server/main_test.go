package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fastygo/app-suite/internal/appschema"
	suiteapp "github.com/fastygo/app-suite/pkg/app"
)

func TestAppSuiteSmokeRoutes(t *testing.T) {
	registry, err := appschema.NewRegistry(appschema.WorkspacesFullProfile())
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}
	server := httptest.NewServer(suiteapp.NewMux(suiteapp.Options{StaticDir: "../../web/static", Registry: registry}))
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
	server := httptest.NewServer(suiteapp.NewMux(suiteapp.Options{StaticDir: "../../web/static", Registry: registry}))
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
