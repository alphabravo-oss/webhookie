package observe

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var (
	Captured   atomic.Int64
	Invalid    atomic.Int64
	Sends      atomic.Int64
	Pruned     atomic.Int64
	SSEClients atomic.Int64
	Stored     atomic.Int64
)

func Metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "webhookie_events_captured_total %d\n", Captured.Load())
	_, _ = fmt.Fprintf(w, "webhookie_events_invalid_total %d\n", Invalid.Load())
	_, _ = fmt.Fprintf(w, "webhookie_send_attempts_total %d\n", Sends.Load())
	_, _ = fmt.Fprintf(w, "webhookie_events_pruned_total %d\n", Pruned.Load())
	_, _ = fmt.Fprintf(w, "webhookie_sse_clients %d\n", SSEClients.Load())
	_, _ = fmt.Fprintf(w, "webhookie_events_stored %d\n", Stored.Load())
}
