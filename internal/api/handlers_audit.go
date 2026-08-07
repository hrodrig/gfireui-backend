package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

const (
	defaultAuditLimit = 100
)

type auditEventResponse struct {
	ID           uuid.UUID       `json:"id"`
	ActorUserID  *uuid.UUID      `json:"actor_user_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *string         `json:"resource_id"`
	IP           *string         `json:"ip"`
	UserAgent    *string         `json:"user_agent"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if s.deps.AuditStore == nil {
		writeError(w, http.StatusInternalServerError, "audit store is not configured")
		return
	}

	limit, offset, err := parseAuditPagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	events, err := s.deps.AuditStore.ListAudit(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list audit events")
		return
	}

	response := make([]auditEventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, newAuditEventResponse(event))
	}

	writeJSON(w, http.StatusOK, response)
}

func parseAuditPagination(r *http.Request) (int, int, error) {
	limit := defaultAuditLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			return 0, 0, errors.New("limit must be a positive integer")
		}
		limit = value
	}

	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
		offset = value
	}

	return limit, offset, nil
}

func newAuditEventResponse(event domain.AuditEvent) auditEventResponse {
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}

	return auditEventResponse{
		ID:           event.ID,
		ActorUserID:  event.ActorUserID,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		IP:           event.IP,
		UserAgent:    event.UserAgent,
		Payload:      payload,
		CreatedAt:    event.CreatedAt,
	}
}
