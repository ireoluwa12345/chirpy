package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/ireoluwa12345/chirpy/internal/auth"
	"github.com/ireoluwa12345/chirpy/internal/database"
)

func (cfg *apiConfig) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var params struct {
		Email    string `json:"email" validate:"required"`
		Password string `json:"password" validate:"required"`
	}
	err := decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	validate := validator.New()
	err = validate.Struct(params)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}

	id := uuid.New()

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}
	params.Password = hashedPassword

	user, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{ID: id, Email: params.Email, Password: hashedPassword})

	resp, err := json.Marshal(map[string]interface{}{
		"id":            user.ID,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleLoginUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	user, err := cfg.db.GetUserByEmail(context.Background(), params.Email)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "No user found"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error fetching user by email:", err)
		w.Write([]byte(`{"error": "couldn't get user}`))
		return
	}

	authenticated, err := auth.VerifyPassword(params.Password, user.Password)

	if err != nil || !authenticated {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "email or password is incorrect"}`))
		return
	}

	expiresIn, err := time.ParseDuration(accessTokenExpiry)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiresIn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	refreshToken, err := auth.MakeRefreshToken()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	expiresAt := time.Now().Add(time.Duration(refreshTokenExpiry) * time.Hour)

	storedRefreshToken, err := cfg.db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "error occurred"}`))
		return
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"id":            user.ID,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
		"token":         jwtToken,
		"refresh_token": storedRefreshToken.Token,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleUpdateUsers(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	user_id, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting validating token: %v", err)))
		return
	}

	var params struct {
		Email    string
		Password string
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&params)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.UpdateUser(context.Background(), database.UpdateUserParams{
		Email:    params.Email,
		Password: hashedPassword,
		ID:       user_id,
	})

	resp, _ := json.Marshal(map[string]interface{}{
		"id":            user.ID,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}
