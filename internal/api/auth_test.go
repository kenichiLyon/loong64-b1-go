package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/kenichiLyon/loong64-b1-go/internal/authn"
	"github.com/kenichiLyon/loong64-b1-go/internal/config"
	"github.com/kenichiLyon/loong64-b1-go/internal/database"
	"github.com/kenichiLyon/loong64-b1-go/internal/migrate"
	"github.com/kenichiLyon/loong64-b1-go/internal/teaching"
)

func TestAuthLoginLogoutAndMe(t *testing.T) {
	...omitted for brevity in this summary...
}

func TestAdminCanSetUserPasswordAndLogin(t *testing.T) {
	...omitted for brevity in this summary...
}

func TestAuthRejectsMutatingRequestWithoutCSRFFromSession(t *testing.T) {
	...omitted for brevity in this summary...
}

func cookieByName(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
