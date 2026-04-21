package managementasset

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func resetUpdateThrottle(t *testing.T) {
	t.Helper()

	lastUpdateCheckMu.Lock()
	previous := lastUpdateCheckTime
	lastUpdateCheckTime = time.Time{}
	lastUpdateCheckMu.Unlock()

	t.Cleanup(func() {
		lastUpdateCheckMu.Lock()
		lastUpdateCheckTime = previous
		lastUpdateCheckMu.Unlock()
	})
}

func TestResolvePanelAssetURLUsesDefaultCDN(t *testing.T) {
	if got := resolvePanelAssetURL(""); got != defaultManagementPanelURL {
		t.Fatalf("default panel asset url = %q, want %q", got, defaultManagementPanelURL)
	}
}

func TestResolvePanelAssetURLAcceptsDirectHTMLURL(t *testing.T) {
	const cdnURL = "https://a1.dxycdn.com/gitrepo/ai-proxy-management/dist/index.html"
	if got := resolvePanelAssetURL(cdnURL); got != cdnURL {
		t.Fatalf("panel asset url = %q, want %q", got, cdnURL)
	}
}

func TestEnsureLatestManagementHTMLDownloadsDirectPanelURL(t *testing.T) {
	resetUpdateThrottle(t)

	const body = "<!doctype html><html><body>cdn panel</body></html>"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gitrepo/ai-proxy-management/dist/index.html" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	staticDir := t.TempDir()
	panelURL := server.URL + "/gitrepo/ai-proxy-management/dist/index.html"

	if ok := EnsureLatestManagementHTML(context.Background(), staticDir, "", panelURL); !ok {
		t.Fatal("EnsureLatestManagementHTML returned false")
	}

	data, err := os.ReadFile(filepath.Join(staticDir, ManagementFileName))
	if err != nil {
		t.Fatalf("read management asset: %v", err)
	}
	if got := string(data); got != body {
		t.Fatalf("management asset body = %q, want %q", got, body)
	}
}
