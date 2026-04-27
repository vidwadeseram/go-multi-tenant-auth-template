package handlers

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeDriver struct {
	pingErr error
}

type fakeConn struct {
	pingErr error
}

func (d *fakeDriver) Open(_ string) (driver.Conn, error) {
	return &fakeConn{pingErr: d.pingErr}, nil
}

func (c *fakeConn) Prepare(_ string) (driver.Stmt, error) { return nil, nil }
func (c *fakeConn) Close() error                          { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)             { return nil, nil }

func (c *fakeConn) Ping(_ context.Context) error { return c.pingErr }

func init() {
	gin.SetMode(gin.TestMode)
}

func newFakeDB(t *testing.T, pingErr error) *sql.DB {
	t.Helper()
	driverName := "fake_" + t.Name()
	sql.Register(driverName, &fakeDriver{pingErr: pingErr})
	db, err := sql.Open(driverName, "")
	if err != nil {
		t.Fatalf("failed to open fake db: %v", err)
	}
	return db
}

func TestHealthHandler_Handle(t *testing.T) {
	tests := []struct {
		name           string
		pingErr        error
		wantStatus     int
		wantBodyStatus string
	}{
		{
			name:           "healthy DB returns 200 ok",
			pingErr:        nil,
			wantStatus:     http.StatusOK,
			wantBodyStatus: "ok",
		},
		{
			name:           "unhealthy DB returns 503 unavailable",
			pingErr:        sql.ErrConnDone,
			wantStatus:     http.StatusServiceUnavailable,
			wantBodyStatus: "unavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newFakeDB(t, tc.pingErr)
			defer db.Close()

			handler := NewHealthHandler(db)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

			handler.Handle(c)

			if w.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, w.Code)
			}

			var body map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to parse response body: %v", err)
			}
			if body["status"] != tc.wantBodyStatus {
				t.Errorf("expected body status %q, got %q", tc.wantBodyStatus, body["status"])
			}
		})
	}
}
