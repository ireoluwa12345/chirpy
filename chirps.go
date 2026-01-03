package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/ireoluwa12345/chirpy/internal/auth"
	"github.com/ireoluwa12345/chirpy/internal/database"
)

func validateChirp(w http.ResponseWriter, r *http.Request) {
	jsonDecoder := json.NewDecoder(r.Body)
	jsonData := struct {
		Body string `json:"body"`
	}{}
	err := jsonDecoder.Decode(&jsonData)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Something went wrong"}`))
		return
	}

	if len(jsonData.Body) > 140 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Chirp is too long"}`))
		return
	}

	forbiddenWords := map[string]bool{
		"kerfuffle": true,
		"sharbert":  true,
		"fornax":    true,
	}

	bodyArray := strings.Split(jsonData.Body, " ")

	for i := 0; i < len(bodyArray); i++ {
		if ok := forbiddenWords[strings.ToLower(bodyArray[i])]; ok {
			bodyArray[i] = "****"
		}
	}

	result := strings.Join(bodyArray, " ")

	resp, _ := json.Marshal(map[string]string{
		"cleaned_body": result,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleCreateChirp(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	param := struct {
		Body string `json:"body"`
	}{}

	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting bearer token: %v", err)))
		return
	}

	user_id, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(fmt.Sprintf("error occurred getting validating token: %v", err)))
		return
	}

	err = decoder.Decode(&param)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't decode json}`))
		return
	}

	chirp, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
		ID:     uuid.New(),
		UserID: user_id,
		Body:   param.Body,
	})

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error creating chirps:", err)
		w.Write([]byte(`{"error": "couldn't create chirp}`))
		return
	}

	resp, err := json.Marshal(chirp)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
}
func (cfg *apiConfig) HandleGetChirps(w http.ResponseWriter, r *http.Request) {
	authorIDString := r.URL.Query().Get("author_id")
	var authorID uuid.UUID

	if authorIDString != "" {
		var err error
		authorID, err = uuid.Parse(authorIDString)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid author ID"}`))
			return
		}
	}

	sortString := r.URL.Query().Get("sort")

	chirps, err := cfg.db.GetChirps(context.Background())

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error fetching chirps:", err)
		w.Write([]byte(`{"error": "couldn't get chirps"}`))
		return
	}

	var filteredChirps []database.Chirp
	for _, dbChirp := range chirps {
		if authorID != uuid.Nil && dbChirp.UserID != authorID {
			continue
		}
		filteredChirps = append(filteredChirps, dbChirp)
	}

	if sortString == "asc" {
		sort.Slice(filteredChirps, func(i, j int) bool {
			return filteredChirps[i].CreatedAt.Before(filteredChirps[j].CreatedAt)
		})
	}

	resp, err := json.Marshal(filteredChirps)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "couldn't marshal json"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (cfg *apiConfig) HandleGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)

	fmt.Println(chirpIDString)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid UUID}`))
		return
	}

	chirp, err := cfg.db.GetChirpByID(context.Background(), chirpID)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "chirp not found"}`))
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		log.Println("Error fetching chirp by ID:", err)
		w.Write([]byte(`{"error": "couldn't get chirp}`))
		return
	}

	resp, err := json.Marshal(chirp)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "couldn't marshal json}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp))
}

func (cfg *apiConfig) HandleDeleteChirps(w http.ResponseWriter, r *http.Request) {
	chirpIDString := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDString)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid UUID"}`))
		return
	}

	userID := r.Context().Value("user_id").(uuid.UUID)

	chirp, err := cfg.db.GetChirpByID(context.Background(), chirpID)

	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "chirp not found"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error fetching chirp by ID:", err)
		w.Write([]byte(`{"error": "couldn't get chirp"}`))
		return
	}

	if chirp.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "you can only delete your own chirps"}`))
		return
	}

	err = cfg.db.DeleteChirp(context.Background(), chirpID)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("Error deleting chirp:", err)
		w.Write([]byte(`{"error": "couldn't delete chirp"}`))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
