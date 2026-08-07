package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/domain"
	"github.com/hrodrig/gfireui-backend/internal/store"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID        uuid.UUID   `json:"id"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	Email     string      `json:"email"`
	Role      domain.Role `json:"role"`
	Enabled   bool        `json:"enabled"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.deps.Store == nil {
		writeError(w, http.StatusInternalServerError, "auth store is not configured")
		return
	}
	if len(s.deps.JWTSecret) == 0 {
		writeError(w, http.StatusInternalServerError, "jwt secret is not configured")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := s.deps.Store.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeLoginAudit(r, nil, email, "failure", "invalid_credentials")
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		s.writeLoginAudit(r, user, email, "failure", "invalid_credentials")
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !user.Enabled {
		s.writeLoginAudit(r, user, email, "failure", "disabled")
		writeError(w, http.StatusForbidden, "user is disabled")
		return
	}

	token, err := auth.IssueToken(s.deps.JWTSecret, s.deps.TokenTTL, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	s.writeLoginAudit(r, user, email, "success", "")
	writeJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User:  newUserResponse(user),
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}

func (s *Server) writeLoginAudit(r *http.Request, user *domain.User, email, result, reason string) {
	if s.deps.Audit == nil {
		return
	}

	payloadMap := map[string]string{
		"email":  email,
		"result": result,
	}
	if reason != "" {
		payloadMap["reason"] = reason
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return
	}

	event := &domain.AuditEvent{
		Action:       "auth.login." + result,
		ResourceType: "user",
		Payload:      payload,
		IP:           optionalString(r.RemoteAddr),
		UserAgent:    optionalString(r.UserAgent()),
	}

	if id, err := uuid.NewV7(); err == nil {
		event.ID = id
	}
	if user != nil {
		actorUserID := user.ID
		resourceID := user.ID.String()
		event.ActorUserID = &actorUserID
		event.ResourceID = &resourceID
	}

	_ = s.deps.Audit.WriteAudit(r.Context(), event)
}

func newUserResponse(user *domain.User) userResponse {
	return userResponse{
		ID:        user.ID,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Email:     user.Email,
		Role:      user.Role,
		Enabled:   user.Enabled,
	}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
