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

type usersListResponse []userResponse

type createUserRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Enabled   *bool  `json:"enabled"`
	Password  string `json:"password"`
}

type updateUserRequest struct {
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Role      *string `json:"role"`
	Enabled   *bool   `json:"enabled"`
}

type updatePasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleUsersList(w http.ResponseWriter, r *http.Request) {
	if s.deps.Store == nil {
		writeError(w, http.StatusInternalServerError, "user store is not configured")
		return
	}

	users, err := s.deps.Store.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	response := make(usersListResponse, 0, len(users))
	for _, user := range users {
		user := user
		response = append(response, newUserResponse(&user))
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleUsersCreate(w http.ResponseWriter, r *http.Request) {
	if s.deps.Store == nil {
		writeError(w, http.StatusInternalServerError, "user store is not configured")
		return
	}

	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	user, payload, password, err := buildCreateUser(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}
	user.PasswordHash = hash

	if err := s.deps.Store.CreateUser(r.Context(), user); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	s.writeUserMutationAudit(r, "users.create", user.ID, payload)
	writeJSON(w, http.StatusCreated, newUserResponse(user))
}

func (s *Server) handleUserGet(w http.ResponseWriter, r *http.Request) {
	if s.deps.Store == nil {
		writeError(w, http.StatusInternalServerError, "user store is not configured")
		return
	}

	user, err := s.loadUserFromPath(r)
	if err != nil {
		s.writeUserLookupError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newUserResponse(user))
}

func (s *Server) handleUserPatch(w http.ResponseWriter, r *http.Request) {
	if s.deps.Store == nil {
		writeError(w, http.StatusInternalServerError, "user store is not configured")
		return
	}

	existing, err := s.loadUserFromPath(r)
	if err != nil {
		s.writeUserLookupError(w, err)
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	payload, err := applyUserPatch(existing, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.deps.Store.UpdateUser(r.Context(), existing); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	s.writeUserMutationAudit(r, "users.update", existing.ID, payload)
	writeJSON(w, http.StatusOK, newUserResponse(existing))
}

func (s *Server) handleUserPassword(w http.ResponseWriter, r *http.Request) {
	if s.deps.Store == nil {
		writeError(w, http.StatusInternalServerError, "user store is not configured")
		return
	}

	user, err := s.loadUserFromPath(r)
	if err != nil {
		s.writeUserLookupError(w, err)
		return
	}

	var req updatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user.PasswordHash = hash
	if err := s.deps.Store.UpdateUser(r.Context(), user); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	s.writeUserMutationAudit(r, "users.password.update", user.ID, map[string]any{
		"user_id": user.ID.String(),
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) loadUserFromPath(r *http.Request) (*domain.User, error) {
	rawID := strings.TrimSpace(r.PathValue("id"))
	if rawID == "" {
		return nil, errors.New("user id is required")
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, errors.New("invalid user id")
	}

	user, err := s.deps.Store.GetUserByID(r.Context(), id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Server) writeUserLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err.Error() == "user id is required" || err.Error() == "invalid user id" {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, "failed to load user")
}

func (s *Server) writeUserMutationAudit(r *http.Request, action string, userID uuid.UUID, payload map[string]any) {
	if s.deps.Audit == nil {
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	event := &domain.AuditEvent{
		Action:       action,
		ResourceType: "user",
		Payload:      body,
	}
	if id, err := uuid.NewV7(); err == nil {
		event.ID = id
	}
	resourceID := userID.String()
	event.ResourceID = &resourceID
	event.IP = optionalString(r.RemoteAddr)
	event.UserAgent = optionalString(r.UserAgent())
	if actor, ok := auth.UserFromContext(r.Context()); ok && actor != nil {
		actorID := actor.ID
		event.ActorUserID = &actorID
	}

	_ = s.deps.Audit.WriteAudit(r.Context(), event)
}

func buildCreateUser(req createUserRequest) (*domain.User, map[string]any, string, error) {
	firstName := strings.TrimSpace(req.FirstName)
	if firstName == "" {
		return nil, nil, "", errors.New("first_name is required")
	}
	lastName := strings.TrimSpace(req.LastName)
	if lastName == "" {
		return nil, nil, "", errors.New("last_name is required")
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return nil, nil, "", errors.New("email is required")
	}
	role := domain.Role(strings.TrimSpace(req.Role))
	if !role.Valid() {
		return nil, nil, "", errors.New("role is invalid")
	}
	if req.Enabled == nil {
		return nil, nil, "", errors.New("enabled is required")
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		return nil, nil, "", errors.New("password is required")
	}

	user := &domain.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Role:      role,
		Enabled:   *req.Enabled,
	}
	payload := map[string]any{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"role":       string(role),
		"enabled":    *req.Enabled,
	}
	return user, payload, password, nil
}

func applyUserPatch(user *domain.User, req updateUserRequest) (map[string]any, error) {
	payload := map[string]any{}

	if req.FirstName != nil {
		value := strings.TrimSpace(*req.FirstName)
		if value == "" {
			return nil, errors.New("first_name must not be empty")
		}
		user.FirstName = value
		payload["first_name"] = value
	}
	if req.LastName != nil {
		value := strings.TrimSpace(*req.LastName)
		if value == "" {
			return nil, errors.New("last_name must not be empty")
		}
		user.LastName = value
		payload["last_name"] = value
	}
	if req.Email != nil {
		value := strings.TrimSpace(*req.Email)
		if value == "" {
			return nil, errors.New("email must not be empty")
		}
		user.Email = value
		payload["email"] = value
	}
	if req.Role != nil {
		role := domain.Role(strings.TrimSpace(*req.Role))
		if !role.Valid() {
			return nil, errors.New("role is invalid")
		}
		user.Role = role
		payload["role"] = string(role)
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
		payload["enabled"] = *req.Enabled
	}

	if len(payload) == 0 {
		return nil, errors.New("at least one field is required")
	}

	return payload, nil
}
