package management

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	proxyconfig "github.com/router-for-me/CLIProxyAPI/v6/internal/config"
)

func TestPublicMonitorAPIKeyMiddlewareAllowsConfiguredKeyWithoutManagementAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&proxyconfig.Config{
		SDKConfig: proxyconfig.SDKConfig{
			APIKeys: proxyconfig.FlexAPIKeyList{"sk-valid"},
		},
	}, "", nil)

	router := gin.New()
	router.GET("/public-monitor", h.PublicMonitorAPIKeyMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/public-monitor?api_key=sk-valid", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPublicMonitorAPIKeyMiddlewareRejectsUnknownKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&proxyconfig.Config{
		SDKConfig: proxyconfig.SDKConfig{
			APIKeys: proxyconfig.FlexAPIKeyList{"sk-valid"},
		},
	}, "", nil)

	router := gin.New()
	router.GET("/public-monitor", h.PublicMonitorAPIKeyMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/public-monitor?api_key=sk-missing", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPublicMonitorAPIKeyMiddlewareAllowsConfigFileEntry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configYaml := []byte(`
auth:
  providers:
    config-api-key:
      api-key-entries:
        - name: Portal Key
          api-key: sk-config-valid
`)
	if err := os.WriteFile(configPath, configYaml, 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	h := NewHandler(&proxyconfig.Config{}, configPath, nil)

	router := gin.New()
	router.GET("/public-monitor", h.PublicMonitorAPIKeyMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/public-monitor?api_key=sk-config-valid", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestPublicMonitorAPIKeyMiddlewareForcesValidatedKeyFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewHandler(&proxyconfig.Config{
		SDKConfig: proxyconfig.SDKConfig{
			APIKeys: proxyconfig.FlexAPIKeyList{"sk-valid"},
		},
	}, "", nil)

	router := gin.New()
	router.GET("/public-monitor", h.PublicMonitorAPIKeyMiddleware(), func(c *gin.Context) {
		filter := h.buildMonitorRecordFilter(c, nil, nil, "")
		if filter.APIKey != "sk-valid" {
			t.Fatalf("public monitor filter used %q", filter.APIKey)
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/public-monitor?api_key=sk-valid&api=sk-other", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", rr.Code, rr.Body.String())
	}
}
