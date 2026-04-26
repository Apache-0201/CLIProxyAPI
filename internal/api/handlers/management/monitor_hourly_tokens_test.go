package management

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestGetMonitorHourlyTokens_UsesRequestedRangeEndAsAnchor(t *testing.T) {
	gin.SetMode(gin.TestMode)

	anchor := time.Now().Local().Add(-24 * time.Hour).Truncate(time.Hour)
	slotTime := anchor.Add(-2 * time.Hour)
	start := anchor.Add(-5 * time.Hour)

	h := newMonitorTestHandler(
		testUsageRecord(slotTime.Add(10*time.Minute), "api-1", "model-a", "source-a", false, 1000, 180),
	)

	rr := executeMonitorRequest(
		h.GetMonitorHourlyTokens,
		"/monitor/hourly-tokens?hours=6&start_time="+url.QueryEscape(start.Format(time.RFC3339))+"&end_time="+url.QueryEscape(anchor.Format(time.RFC3339)),
	)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d, body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Hours       []string `json:"hours"`
		TotalTokens []int64  `json:"total_tokens"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}

	expectedLastSlot := anchor.Format("2006-01-02T15:04:05-07:00")
	if got := resp.Hours[len(resp.Hours)-1]; got != expectedLastSlot {
		t.Fatalf("unexpected slot anchor: got %s want %s", got, expectedLastSlot)
	}

	expectedSlot := slotTime.Format("2006-01-02T15:04:05-07:00")
	slotIndex := -1
	for i, slot := range resp.Hours {
		if slot == expectedSlot {
			slotIndex = i
			break
		}
	}
	if slotIndex < 0 {
		t.Fatalf("hour slot not found: %s", expectedSlot)
	}
	if resp.TotalTokens[slotIndex] != 30 {
		t.Fatalf("unexpected total tokens for slot: got %d want 30", resp.TotalTokens[slotIndex])
	}
}
