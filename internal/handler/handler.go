package handler


import (
	"encoding/json"
	"errors"
	"net/http"

	"ex8ed/pullreq-assigner/internal/entity"
	"ex8ed/pullreq-assigner/internal/service"
	"ex8ed/pullreq-assigner/internal/storage"
)


type Handler struct {
	svc *service.Service
}


func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}


func (h *Handler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		json.NewEncoder(w).Encode(payload)
	}
}

func (h *Handler) respondError(w http.ResponseWriter, err error) {
	statusCode := http.StatusInternalServerError
	msg := "internal server error"
	appCode := "ERROR"
	switch {
	case errors.Is(err, storage.ErrNotFound):
		statusCode = http.StatusNotFound
		appCode = "NOT_FOUND"
		msg = "resource not found"

	case errors.Is(err, storage.ErrAlreadyExists):
		statusCode = http.StatusConflict
		appCode = "RESOURCE_EXISTS"
		msg = "resource already exists"

	case errors.Is(err, service.ErrPRMerged):
		statusCode = http.StatusConflict
		appCode = "PR_MERGED"
		msg = "cannot edit merged PR"

	case errors.Is(err, service.ErrNotAssigned):
		statusCode = http.StatusConflict
		appCode = "NOT_ASSIGNED"
		msg = "reviewer is not assigned to this PR"

	case errors.Is(err, service.ErrNoCandidates):
		statusCode = http.StatusConflict
		appCode = "NO_CANDIDATE"
		msg = "no active replacement candidate in team"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    appCode,
			"message": msg,
		},
	})
}


func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// -------------------------------------------------------------------
// TEAMS
// -------------------------------------------------------------------

// POST /team/add
func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req entity.Team
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := h.svc.CreateTeam(r.Context(), req); err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"team": req,
	})
}

// GET /team/get?team_name=...
func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("team_name")
	if name == "" {
		http.Error(w, "missing team_name", http.StatusBadRequest)
		return
	}

	team, err := h.svc.GetTeam(r.Context(), name)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, team)
}

// -------------------------------------------------------------------
// USERS
// -------------------------------------------------------------------

// POST /users/setIsActive
func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if err := h.svc.SetUserActive(r.Context(), req.UserID, req.IsActive); err != nil {
		h.respondError(w, err)
		return
	}

	user, err := h.svc.GetUser(r.Context(), req.UserID)
	if err != nil {
		h.respondError(w, err)
	}
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// GET /users/getReview?user_id=...
func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	prs, err := h.svc.GetUserReviews(r.Context(), userID)
	if err != nil {
		h.respondError(w, err)
		return
	}

	if prs == nil {
		prs = []entity.PullRequest{}
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

// -------------------------------------------------------------------
// PULL REQUESTS
// -------------------------------------------------------------------

// POST /pullRequest/create
func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID       string `json:"pull_request_id"`
		Name     string `json:"pull_request_name"`
		AuthorID string `json:"author_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	pr, err := h.svc.CreatePR(r.Context(), req.ID, req.Name, req.AuthorID)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"pr": pr,
	})
}

// POST /pullRequest/merge
func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"pull_request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	pr, err := h.svc.MergePR(r.Context(), req.ID)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr": pr,
	})
}

// POST /pullRequest/reassign
func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PRID      string `json:"pull_request_id"`
		OldUserID string `json:"old_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	pr, newID, err := h.svc.ReassignReviewer(r.Context(), req.PRID, req.OldUserID)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          pr,
		"replaced_by": newID,
	})
}