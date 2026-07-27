package adminapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminapi"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/adminauth"
	"github.com/storesace-cv/bwb-modulo-fiscal/internal/notify/smtp"
)

type stubMailer struct {
	configured bool
	status     smtp.DeliveryStatus
	err        error
}

func (s stubMailer) Configured() bool { return s.configured }
func (s stubMailer) SendAdminTest(context.Context, string) (smtp.DeliveryStatus, error) {
	return s.status, s.err
}

func TestNotificationTestOwnerOnlyAndStatuses(t *testing.T) {
	t.Parallel()
	owner := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "owner-1", Roles: []adminauth.Role{adminauth.RoleOwner},
	}}
	admin := adminauth.StaticAuthenticator{Claims: adminauth.Claims{
		Subject: "admin-1", Roles: []adminauth.Role{adminauth.RoleAdmin},
	}}

	t.Run("admin_forbidden", func(t *testing.T) {
		mux := http.NewServeMux()
		adminapi.Mount(mux, admin, &adminapi.Handler{
			Mailer: stubMailer{configured: true, status: smtp.DeliveryStatus{Status: "sent"}},
		})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/ops/notifications/test", nil))
		if rr.Code != http.StatusForbidden {
			t.Fatalf("want 403 got %d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("not_configured", func(t *testing.T) {
		mux := http.NewServeMux()
		adminapi.Mount(mux, owner, &adminapi.Handler{Mailer: stubMailer{configured: false}})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/ops/notifications/test", nil))
		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("want 503 got %d body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("sent", func(t *testing.T) {
		mux := http.NewServeMux()
		adminapi.Mount(mux, owner, &adminapi.Handler{
			Mailer: stubMailer{configured: true, status: smtp.DeliveryStatus{
				Status: "sent", TLSMode: "implicit", Port: 465, ToDomain: "example.com",
			}},
		})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/ops/notifications/test", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("want 200 got %d body=%s", rr.Code, rr.Body.String())
		}
		var st smtp.DeliveryStatus
		if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
			t.Fatal(err)
		}
		if st.Status != "sent" || st.Port != 465 {
			t.Fatalf("unexpected %+v", st)
		}
	})

	t.Run("send_failed_sanitized", func(t *testing.T) {
		mux := http.NewServeMux()
		adminapi.Mount(mux, owner, &adminapi.Handler{
			Mailer: stubMailer{
				configured: true,
				status:     smtp.DeliveryStatus{Status: "failed", Reason: "smtp_auth_failed"},
				err:        errors.New("auth failed with password=should-not-leak"),
			},
		})
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/admin/v1/ops/notifications/test", nil))
		if rr.Code != http.StatusBadGateway {
			t.Fatalf("want 502 got %d body=%s", rr.Code, rr.Body.String())
		}
		body := rr.Body.String()
		if strings.Contains(body, "should-not-leak") || strings.Contains(body, "password=") {
			t.Fatalf("secret leaked in body: %s", body)
		}
	})
}
