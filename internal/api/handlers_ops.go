package api

import (
	"net/http"
	"time"

	"github.com/hrodrig/gfireui-backend/internal/gfire"
)

var opsJobStates = []string{"pending", "processing", "succeeded", "failed", "dead"}

type opsSummaryResponse struct {
	JobsByState map[string]int   `json:"jobs_by_state"`
	Queues      []gfire.QueueSummary `json:"queues"`
	GeneratedAt time.Time        `json:"generated_at"`
}

func (s *Server) handleOpsSummary(w http.ResponseWriter, r *http.Request) {
	if s.deps.GFire == nil {
		writeError(w, http.StatusInternalServerError, "gfire client is not configured")
		return
	}

	response := opsSummaryResponse{
		JobsByState: make(map[string]int, len(opsJobStates)),
		Queues:      []gfire.QueueSummary{},
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

	writeJSON(w, http.StatusOK, response)
}
