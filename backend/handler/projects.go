package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhinav2712/taskflow-abhinav/middleware"
	"github.com/abhinav2712/taskflow-abhinav/model"
	"github.com/abhinav2712/taskflow-abhinav/store"
)

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type projectListResponse struct {
	Projects []model.Project `json:"projects"`
}

type projectDetailResponse struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	OwnerID     uuid.UUID    `json:"owner_id"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Tasks       []model.Task `json:"tasks"`
}

func ListProjects(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		callerID := middleware.UserIDFromCtx(r.Context())

		projects, err := store.ListProjects(r.Context(), pool, callerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusOK, projectListResponse{Projects: projects})
	}
}

func CreateProject(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createProjectRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fields := map[string]string{}
		if strings.TrimSpace(req.Name) == "" {
			fields["name"] = "is required"
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		callerID := middleware.UserIDFromCtx(r.Context())
		project, err := store.CreateProject(r.Context(), pool, strings.TrimSpace(req.Name), req.Description, callerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusCreated, project)
	}
}

func GetProject(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid project id")
			return
		}

		project, err := store.GetProject(r.Context(), pool, projectID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		callerID := middleware.UserIDFromCtx(r.Context())
		if !canAccessProject(project, callerID) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		encode(w, http.StatusOK, projectDetailResponse{
			ID:          project.ID,
			Name:        project.Name,
			Description: project.Description,
			OwnerID:     project.OwnerID,
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
			Tasks:       project.Tasks,
		})
	}
}

func UpdateProject(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid project id")
			return
		}

		var req updateProjectRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fields := map[string]string{}
		if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
			fields["name"] = "is required"
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		project, err := store.GetProject(r.Context(), pool, projectID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		callerID := middleware.UserIDFromCtx(r.Context())
		if project.OwnerID != callerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		if req.Name == nil && req.Description == nil {
			project.Tasks = nil
			encode(w, http.StatusOK, project)
			return
		}

		if req.Name != nil {
			trimmedName := strings.TrimSpace(*req.Name)
			req.Name = &trimmedName
		}

		project, err = store.UpdateProject(r.Context(), pool, projectID, req.Name, req.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusOK, project)
	}
}

func DeleteProject(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid project id")
			return
		}

		project, err := store.GetProject(r.Context(), pool, projectID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		callerID := middleware.UserIDFromCtx(r.Context())
		if project.OwnerID != callerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		if err := store.DeleteProject(r.Context(), pool, projectID); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func canAccessProject(project model.Project, callerID uuid.UUID) bool {
	if project.OwnerID == callerID {
		return true
	}

	for _, task := range project.Tasks {
		if task.AssigneeID != nil && *task.AssigneeID == callerID {
			return true
		}
	}

	return false
}
