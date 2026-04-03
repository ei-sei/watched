package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ei-sei/brsti/internal/auth"
	"github.com/ei-sei/brsti/internal/config"
	"github.com/ei-sei/brsti/internal/db"
	"github.com/ei-sei/brsti/internal/handler"
	"github.com/ei-sei/brsti/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	// Repositories
	userRepo    := repository.NewUserRepo(pool)
	mediaRepo   := repository.NewMediaRepo(pool)
	episodeRepo := repository.NewEpisodeRepo(pool)
	chapterRepo := repository.NewChapterRepo(pool)
	listRepo    := repository.NewListRepo(pool)

	// Handlers
	authH   := handler.NewAuthHandler(userRepo, cfg)
	userH   := handler.NewUserHandler(userRepo, cfg)
	mediaH  := handler.NewMediaHandler(mediaRepo, episodeRepo, chapterRepo)
	listH   := handler.NewListHandler(listRepo, mediaRepo)
	searchH := handler.NewSearchHandler(cfg)
	statsH  := handler.NewStatsHandler(mediaRepo)
	shareH  := handler.NewShareHandler(listRepo)
	importH := handler.NewImportHandler(mediaRepo, cfg)

	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CORSOrigins},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok")) //nolint:errcheck
	})

	// Public share routes — no auth
	r.Get("/share/lists/{id}", shareH.GetList)

	// Public auth routes
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.Post("/logout", authH.Logout)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate(cfg.JWTSecret))

		// Current user
		r.Get("/users/me", userH.Me)
		r.Patch("/users/me", userH.UpdateMe)
		r.Put("/users/me/password", userH.ChangePassword)

		// Media
		r.Route("/media", func(r chi.Router) {
			r.Get("/", mediaH.List)
			r.Post("/", mediaH.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", mediaH.Get)
				r.Patch("/", mediaH.Update)
				r.Delete("/", mediaH.Delete)

				// TV episodes
				r.Get("/episodes", mediaH.ListEpisodes)
				r.Put("/episodes", mediaH.UpsertEpisode)
				r.Delete("/episodes/{epID}", mediaH.DeleteEpisode)

				// Book chapters
				r.Get("/chapters", mediaH.ListChapters)
				r.Put("/chapters", mediaH.UpsertChapter)
				r.Delete("/chapters/{chID}", mediaH.DeleteChapter)
			})
		})

		// Lists
		r.Route("/lists", func(r chi.Router) {
			r.Get("/", listH.List)
			r.Post("/", listH.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", listH.Get)
				r.Patch("/", listH.Update)
				r.Delete("/", listH.Delete)
				r.Post("/items", listH.AddItem)
				r.Delete("/items/{mediaID}", listH.RemoveItem)
				r.Put("/items/order", listH.ReorderItems)
			})
		})

		// Search & Stats
		r.Get("/search", searchH.Search)
		r.Get("/stats", statsH.Get)

		// Import
		r.Post("/import/mal/file", importH.ImportXML)
		r.Post("/import/mal/username", importH.ImportUsername)

		// Admin
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin)
			r.Get("/admin/users", userH.AdminList)
			r.Patch("/admin/users/{id}/flags", userH.AdminUpdateFlags)
			r.Post("/admin/invites", userH.AdminCreateInvite)
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("stopped")
}
