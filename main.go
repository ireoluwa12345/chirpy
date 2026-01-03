package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/ireoluwa12345/chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	hits      atomic.Int32
	db        *database.Queries
	jwtSecret string
}

func main() {
	port := "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error occurred: %v", err)
	}
	dbQueries := database.New(db)

	mux := http.NewServeMux()
	apiMux := http.NewServeMux()
	adminMux := http.NewServeMux()

	apiCfg := &apiConfig{
		hits:      atomic.Int32{},
		db:        dbQueries,
		jwtSecret: jwtSecret,
	}

	fileServer := http.StripPrefix("/app/", http.FileServer(http.Dir("./")))

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServer))
	apiMux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	apiMux.HandleFunc("POST /validate_chirp", validateChirp)
	apiMux.HandleFunc("POST /users", apiCfg.HandleCreateUser)
	apiMux.HandleFunc("PUT /users", apiCfg.HandleUpdateUsers)
	apiMux.HandleFunc("POST /login", apiCfg.HandleLoginUser)
	apiMux.HandleFunc("POST /chirps", apiCfg.HandleCreateChirp)
	apiMux.HandleFunc("GET /chirps", apiCfg.HandleGetChirps)
	apiMux.HandleFunc("GET /chirps/{chirpID}", apiCfg.HandleGetChirpByID)
	apiMux.HandleFunc("POST /refresh", apiCfg.HandleRefresh)
	apiMux.HandleFunc("POST /revoke", apiCfg.handleRevoke)
	apiMux.Handle("DELETE /chirps/{chirpID}", apiCfg.authorize(http.HandlerFunc(apiCfg.HandleDeleteChirps)))

	adminMux.HandleFunc("GET /metrics", apiCfg.fileServerHits)
	adminMux.HandleFunc("POST /reset", apiCfg.fileServerReset)

	mux.Handle("/api/", http.StripPrefix("/api", apiMux))
	mux.Handle("/admin/", http.StripPrefix("/admin", adminMux))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	srv.ListenAndServe()
}
