package authstack

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	frameworkauth "github.com/fastygo/framework/pkg/auth"
	"github.com/fastygo/platform/pkg/contracts"
)

func TestCookieSessionCarriesOnlyPrincipalAndProfileClaims(t *testing.T) {
	session := frameworkauth.CookieSession[contracts.SessionClaims]{
		Name:     "suite_session",
		Path:     "/",
		Secret:   "01234567890123456789012345678901",
		TTL:      time.Hour,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	recorder := httptest.NewRecorder()
	claims := contracts.SessionClaims{PrincipalID: "user-1", ProfileID: "suite"}
	if err := session.Issue(recorder, claims); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/go-admin", nil)
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}
	got, ok := session.Read(request)
	if !ok {
		t.Fatalf("expected session claims")
	}
	if got != claims {
		t.Fatalf("claims = %#v, want %#v", got, claims)
	}

	cookie := recorder.Result().Cookies()[0]
	payload, _, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		t.Fatalf("expected signed cookie payload")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(decoded), "capability") || strings.Contains(string(decoded), "workspace") {
		t.Fatalf("session payload must not embed capabilities or workspace grants: %s", decoded)
	}
}
