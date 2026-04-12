package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/abhinav2712/taskflow-abhinav/model"
	"github.com/abhinav2712/taskflow-abhinav/store"
)

const bcryptCost = 12

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}

func Register(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registerRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fields := map[string]string{}
		if strings.TrimSpace(req.Name) == "" {
			fields["name"] = "is required"
		}
		if strings.TrimSpace(req.Email) == "" {
			fields["email"] = "is required"
		} else if !strings.Contains(req.Email, "@") {
			fields["email"] = "must be a valid email"
		}
		if strings.TrimSpace(req.Password) == "" {
			fields["password"] = "is required"
		} else if len(strings.TrimSpace(req.Password)) < 8 {
			fields["password"] = "must be at least 8 characters"
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		if _, err := store.GetUserByEmail(r.Context(), pool, req.Email); err == nil {
			writeError(w, http.StatusConflict, "email already in use")
			return
		} else if !errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		hashedPassword, err := hashPassword(req.Password)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		user, err := store.CreateUser(r.Context(), pool, req.Name, req.Email, hashedPassword)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				writeError(w, http.StatusConflict, "email already in use")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		token, err := generateToken(user.ID, user.Email, jwtSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusCreated, authResponse{
			Token: token,
			User:  user,
		})
	}
}

func Login(pool *pgxpool.Pool, jwtSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := decode(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		fields := map[string]string{}
		if strings.TrimSpace(req.Email) == "" {
			fields["email"] = "is required"
		}
		if strings.TrimSpace(req.Password) == "" {
			fields["password"] = "is required"
		}
		if len(fields) > 0 {
			writeValidationError(w, fields)
			return
		}

		user, err := store.GetUserByEmail(r.Context(), pool, req.Email)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		if err := comparePassword(user.Password, req.Password); err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token, err := generateToken(user.ID, user.Email, jwtSecret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		encode(w, http.StatusOK, authResponse{
			Token: token,
			User:  user,
		})
	}
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func comparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func generateToken(userID uuid.UUID, email, secret string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID.String(),
		"email":   email,
		"iat":     now.Unix(),
		"exp":     now.Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(secret))
}
