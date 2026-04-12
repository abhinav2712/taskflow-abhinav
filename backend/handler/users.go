package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhinav2712/taskflow-abhinav/model"
	"github.com/abhinav2712/taskflow-abhinav/store"
)

type usersListResponse struct {
	Users []model.User `json:"users"`
}

func ListUsers(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := store.ListUsers(r.Context(), pool)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusOK, usersListResponse{Users: users})
	}
}
