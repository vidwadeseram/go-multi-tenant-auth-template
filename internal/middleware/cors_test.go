package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func TestCORSMiddleware_SetsHeaders(t *testing.T) {
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	tests := []struct {
		header string
		want   string
	}{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS"},
		{"Access-Control-Allow-Headers", "Origin, Content-Type, Authorization"},
	}

	for _, tc := range tests {
		got := w.Header().Get(tc.header)
		if got != tc.want {
			t.Errorf("header %s: expected %q, got %q", tc.header, tc.want, got)
		}
	}
}

func TestCORSMiddleware_PreflightOptionsReturns204(t *testing.T) {
	router := gin.New()
	router.Use(corsMiddleware())
	router.OPTIONS("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS preflight, got %d", w.Code)
	}
}
