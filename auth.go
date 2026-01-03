package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ireoluwa12345/chirpy/internal/auth"
)

func (cfg *apiConfig) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting bearer token: %v", err)))
		return
	}

	refreshToken, err := cfg.db.CheckRefreshToken(context.Background(), bearerToken)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	expiresIn, err := time.ParseDuration(accessTokenExpiry)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	accessToken, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, expiresIn)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"token": accessToken,
	})

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *apiConfig) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting bearer token: %v", err)))
		return
	}

	err = cfg.db.RevokeRefreshToken(context.Background(), bearerToken)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
