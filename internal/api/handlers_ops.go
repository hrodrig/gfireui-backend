package api

import (
	"net/http"
	"time"

	"github.com/hrodrig/gfireui-backend/internal/gfire"
	"github.com/hrodrig/gfireui-backend/internal/version"
)

// Canonical GFire job states (engine filter strings).
var opsJobStates = []string{
	"Enqueued",
	"Scheduled",
	"Processing",
	"Succeeded",
	"Failed",
	"Dead",
	"Cancelled",
	"Deleted",
	"Awaiting",
}

type opsVersionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Commit  string `json:"commit,omitempty"`
	URL     string `json:"url"`
}

type opsSummaryResponse struct {
	JobsByState    map[string]int       `json:"jobs_by_state"`
	Queues         []gfire.QueueSummary `json:"queues"`
	ServersCount   int                  `json:"servers_count"`
	RecurringCount int                  `json:"recurring_count"`
	Versions       []opsVersionInfo     `json:"versions"`
	GeneratedAt    time.Time            `json:"generated_at"`
}

func (s *Server) handleOpsSummary(w http.ResponseWriter, r *http.Request) {
	if s.deps.GFire == nil {
		writeError(w, http.StatusInternalServerError, "gfire client is not configured")
		return
	}

	response := opsSummaryResponse{
		JobsByState: make(map[string]int, len(opsJobStates)),
		Queues:      []gfire.QueueSummary{},
		Versions: []opsVersionInfo{
			{
				Name:    "gfireui-backend",
				Version: version.Version,
				Commit:  version.Commit,
				URL:     "https://github.com/hrodrig/gfireui-backend",
			},
			{
				Name: "gfire",
				URL:  "https://github.com/hrodrig/gfire",
			},
		},
		GeneratedAt: time.Now().UTC(),
	}
	for _, state := range opsJobStates {
		response.JobsByState[state] = 0
	}

	if queues, err := s.deps.GFire.ListQueues(r.Context()); err == nil {
		response.Queues = queues
	}
	for _, state := range opsJobStates {
		if count, err := s.deps.GFire.CountJobsByState(r.Context(), state); err == nil {
			response.JobsByState[state] = count
		}
	}
	if n, err := s.deps.GFire.CountServers(r.Context()); err == nil {
		response.ServersCount = n
	}
	if n, err := s.deps.GFire.CountRecurring(r.Context()); err == nil {
		response.RecurringCount = n
	}
	if info, err := s.deps.GFire.FetchVersion(r.Context()); err == nil {
		response.Versions[1].Version = info.Version
		response.Versions[1].Commit = info.Commit
	}

	writeJSON(w, http.StatusOK, response)
}
