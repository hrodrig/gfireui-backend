package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hrodrig/gfireui-backend/internal/auth"
	"github.com/hrodrig/gfireui-backend/internal/domain"
)

var hopByHopHeaders = map[string]struct{}{
	"Connection":          {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func (s *Server) handleGFireProxy(w http.ResponseWriter, r *http.Request) {
	if s.deps.GFire == nil {
		writeError(w, http.StatusInternalServerError, "gfire client is not configured")
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok || user == nil {
		writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
		return
	}

	allowedRoles, supported := allowedGFireProxyRoles(r.Method)
	if !supported {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !hasAnyRole(user.Role, allowedRoles...) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	upstreamPath := gfireProxyPath(r)

	var body io.Reader
	if gfireMethodHasBody(r.Method) {
		body = r.Body
	}

	resp, err := s.deps.GFire.Do(r.Context(), r.Method, upstreamPath, body)
	if err != nil {
		if gfireMethodIsMutating(r.Method) {
			s.writeGFireProxyAudit(r, user, upstreamPath, http.StatusBadGateway)
		}
		writeError(w, http.StatusBadGateway, "gfire upstream request failed")
		return
	}
	defer resp.Body.Close()

	copyProxyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, resp.Body)
	}

	if gfireMethodIsMutating(r.Method) {
		s.writeGFireProxyAudit(r, user, upstreamPath, resp.StatusCode)
	}
}

func allowedGFireProxyRoles(method string) ([]domain.Role, bool) {
	switch method {
	case http.MethodGet, http.MethodHead:
		return []domain.Role{
			domain.RoleAdministrator,
			domain.RoleOperator,
			domain.RoleAuditor,
		}, true
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return []domain.Role{
			domain.RoleAdministrator,
			domain.RoleOperator,
		}, true
	default:
		return nil, false
	}
}

func gfireMethodIsMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func gfireMethodHasBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func gfireProxyPath(r *http.Request) string {
	path := strings.TrimSpace(r.PathValue("path"))
	if path == "" {
		path = "/"
	} else {
		path = "/" + strings.TrimLeft(path, "/")
	}
	if r.URL.RawQuery == "" {
		return path
	}
	return path + "?" + r.URL.RawQuery
}

func copyProxyHeaders(dst, src http.Header) {
	for key, values := range src {
		if shouldSkipProxyHeader(key, src) {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func shouldSkipProxyHeader(key string, headers http.Header) bool {
	canonicalKey := http.CanonicalHeaderKey(key)
	if _, ok := hopByHopHeaders[canonicalKey]; ok {
		return true
	}

	for _, token := range connectionHeaderTokens(headers) {
		if strings.EqualFold(token, canonicalKey) {
			return true
		}
	}
	return false
}

func connectionHeaderTokens(headers http.Header) []string {
	values := headers.Values("Connection")
	if len(values) == 0 {
		return nil
	}

	tokens := make([]string, 0)
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				tokens = append(tokens, token)
			}
		}
	}
	return tokens
}

func (s *Server) writeGFireProxyAudit(r *http.Request, user *domain.User, upstreamPath string, statusCode int) {
	if s.deps.Audit == nil || user == nil {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"method":      r.Method,
		"path":        upstreamPath,
		"status_code": statusCode,
	})
	if err != nil {
		return
	}

	event := &domain.AuditEvent{
		Action:       "gfire.proxy",
		ResourceType: "gfire",
		Payload:      payload,
		IP:           optionalString(r.RemoteAddr),
		UserAgent:    optionalString(r.UserAgent()),
	}
	if id, err := uuid.NewV7(); err == nil {
		event.ID = id
	}

	actorID := user.ID
	resourceID := strings.TrimSpace(upstreamPath)
	if resourceID == "" {
		resourceID = "/"
	}
	event.ActorUserID = &actorID
	event.ResourceID = &resourceID

	_ = s.deps.Audit.WriteAudit(r.Context(), event)
}
