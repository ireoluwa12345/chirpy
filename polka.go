package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/ireoluwa12345/chirpy/internal/auth"
)

func (cfg *apiConfig) HandlePolkaWebhook(w http.ResponseWriter, r *http.Request) {
	apiKey, err := auth.GetAPIKey(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if cfg.polkaKey != apiKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var params struct {
		Event string            `json:"event"`
		Data  map[string]string `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&params)

	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(params.Data["user_id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := cfg.db.UpgradeUser(context.Background(), userID)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, _ := json.Marshal(map[string]any{
		"id":            user.ID,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"email":         user.Email,
		"is_chirpy_red": user.IsChirpyRed,
	})

	w.WriteHeader(http.StatusNoContent)
	w.Write(resp)

}
