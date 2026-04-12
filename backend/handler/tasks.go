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

var validTaskStatuses = map[string]bool{
	"todo":        true,
	"in_progress": true,
	"done":        true,
}

var validTaskPriorities = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

type createTaskRequest struct {
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Priority    string     `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	DueDate     *string    `json:"due_date"`
}

type updateTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	Priority    *string    `json:"priority"`
	AssigneeID  *uuid.UUID `json:"assignee_id"`
	DueDate     *string    `json:"due_date"`
}

type taskListResponse struct {
	Tasks []model.Task `json:"tasks"`
}

func ListTasks(pool *pgxpool.Pool) http.HandlerFunc {
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

		var statusFilter *string
		if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
			if !validTaskStatuses[status] {
				writeValidationError(w, map[string]string{"status": "must be one of todo, in_progress, done"})
				return
			}
			statusFilter = &status
		}

		var assigneeFilter *uuid.UUID
		if assignee := strings.TrimSpace(r.URL.Query().Get("assignee")); assignee != "" {
			parsedAssigneeID, err := uuid.Parse(assignee)
			if err != nil {
				writeValidationError(w, map[string]string{"assignee": "must be a valid UUID"})
				return
			}
			assigneeFilter = &parsedAssigneeID
		}

		tasks, err := store.ListTasks(r.Context(), pool, projectID, statusFilter, assigneeFilter)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusOK, taskListResponse{Tasks: tasks})
	}
}

func CreateTask(pool *pgxpool.Pool) http.HandlerFunc {
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

		var req createTaskRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fields := map[string]string{}
		if strings.TrimSpace(req.Title) == "" {
			fields["title"] = "is required"
		}
		if strings.TrimSpace(req.Priority) == "" {
			fields["priority"] = "is required"
		} else if !validTaskPriorities[strings.TrimSpace(req.Priority)] {
			fields["priority"] = "must be one of low, medium, high"
		}
		if req.DueDate != nil {
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(*req.DueDate)); err != nil {
				fields["due_date"] = "must be in YYYY-MM-DD format"
			}
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		input := store.CreateTaskInput{
			Title:       strings.TrimSpace(req.Title),
			Description: req.Description,
			Priority:    strings.TrimSpace(req.Priority),
			AssigneeID:  req.AssigneeID,
			DueDate:     req.DueDate,
		}

		task, err := store.CreateTask(r.Context(), pool, projectID, callerID, input)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusCreated, task)
	}
}

func UpdateTask(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}

		existingTask, err := store.GetTask(r.Context(), pool, taskID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		project, err := store.GetProject(r.Context(), pool, existingTask.ProjectID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		callerID := middleware.UserIDFromCtx(r.Context())

		// User can update task only if:
		// they created the task
		// OR
		// they own the project
		if existingTask.CreatorID != callerID && project.OwnerID != callerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		var req updateTaskRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fields := map[string]string{}
		if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
			fields["title"] = "is required"
		}
		if req.Status != nil {
			trimmed := strings.TrimSpace(*req.Status)
			if !validTaskStatuses[trimmed] {
				fields["status"] = "must be one of todo, in_progress, done"
			} else {
				req.Status = &trimmed
			}
		}
		if req.Priority != nil {
			trimmed := strings.TrimSpace(*req.Priority)
			if !validTaskPriorities[trimmed] {
				fields["priority"] = "must be one of low, medium, high"
			} else {
				req.Priority = &trimmed
			}
		}
		if req.DueDate != nil {
			trimmed := strings.TrimSpace(*req.DueDate)
			if _, err := time.Parse("2006-01-02", trimmed); err != nil {
				fields["due_date"] = "must be in YYYY-MM-DD format"
			} else {
				req.DueDate = &trimmed
			}
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		if req.Title == nil && req.Description == nil && req.Status == nil && req.Priority == nil && req.AssigneeID == nil && req.DueDate == nil {
			encode(w, http.StatusOK, existingTask)
			return
		}

		if req.Title != nil {
			trimmed := strings.TrimSpace(*req.Title)
			req.Title = &trimmed
		}

		task, err := store.UpdateTask(r.Context(), pool, taskID, store.UpdateTaskInput{
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			Priority:    req.Priority,
			AssigneeID:  req.AssigneeID,
			DueDate:     req.DueDate,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusOK, task)
	}
}

func DeleteTask(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid task id")
			return
		}

		task, err := store.GetTask(r.Context(), pool, taskID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		project, err := store.GetProject(r.Context(), pool, task.ProjectID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusNotFound, "not found")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		callerID := middleware.UserIDFromCtx(r.Context())
		if task.CreatorID != callerID && project.OwnerID != callerID {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		if err := store.DeleteTask(r.Context(), pool, taskID); err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
