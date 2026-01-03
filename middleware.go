package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ireoluwa12345/chirpy/internal/auth"
)

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.hits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		ctx := context.WithValue(r.Context(), "user_id", user_id)

		reqWithData := r.WithContext(ctx)

		next.ServeHTTP(w, reqWithData)
	})
}
