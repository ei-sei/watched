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
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: .env load failed: %v", err)
	}

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
	rewatchRepo := repository.NewRewatchRepo(pool)
	portalRepo  := repository.NewPortalRepo(pool)
	flagsRepo   := repository.NewFlagsRepo(pool)

	// Handlers
	authH   := handler.NewAuthHandler(userRepo, cfg)
	userH   := handler.NewUserHandler(userRepo, mediaRepo, flagsRepo, cfg)
	mediaH  := handler.NewMediaHandler(mediaRepo, episodeRepo, chapterRepo, cfg.TMDBKey)
	listH   := handler.NewListHandler(listRepo, mediaRepo)
	searchH  := handler.NewSearchHandler(cfg)
	statsH    := handler.NewStatsHandler(mediaRepo)
	shareH    := handler.NewShareHandler(listRepo)
	importH   := handler.NewImportHandler(mediaRepo, episodeRepo, cfg)
	healthH   := handler.NewHealthHandler(cfg)
	trendingH := handler.NewTrendingHandler(cfg)
	rewatchH  := handler.NewRewatchHandler(rewatchRepo, mediaRepo)
	portalH   := handler.NewPortalHandler(portalRepo)
	flagsH    := handler.NewFlagsHandler(flagsRepo)

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

	// Public routes — no auth
	r.Get("/share/lists/{id}", shareH.GetList)
	r.Get("/u/{username}", userH.PublicProfile)

	// Public auth routes
	r.Route("/auth", func(r chi.Router) {
		r.With(auth.RateLimit(10, 5)).Post("/register", authH.Register)
		r.With(auth.RateLimit(10, 5)).Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.Post("/logout", authH.Logout)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate(cfg.JWTSecret))
		r.Use(auth.RateLimit(300, 50)) // 300 req/min, burst of 50

		// Current user
		r.Get("/users/me", userH.Me)
		r.Patch("/users/me", userH.UpdateMe)
		r.Put("/users/me/password", userH.ChangePassword)

		// Media
		r.Route("/media", func(r chi.Router) {
			r.Get("/", mediaH.List)
			r.Post("/", mediaH.Create)
			r.Delete("/", mediaH.ClearByType)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", mediaH.Get)
				r.Patch("/", mediaH.Update)
				r.Delete("/", mediaH.Delete)
				r.Post("/refresh", mediaH.RefreshFromTMDB)

				// TV episodes
				r.Get("/episodes", mediaH.ListEpisodes)
				r.Get("/episode-stamps", mediaH.ListEpisodeStamps)
				r.Put("/episodes", mediaH.UpsertEpisode)
				r.Put("/episodes/progress", mediaH.SetSeasonProgress)
				r.Delete("/episodes/{epID}", mediaH.DeleteEpisode)

				// Book chapters
				r.Get("/chapters", mediaH.ListChapters)
				r.Put("/chapters", mediaH.UpsertChapter)
				r.Delete("/chapters/{chID}", mediaH.DeleteChapter)
				r.Post("/chapters/import", mediaH.ImportChapters)

				// Rewatches
				r.Get("/rewatches", rewatchH.List)
				r.Post("/rewatches", rewatchH.Create)
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

		// Search — stricter limit since each request fans out to 3-4 external APIs
		r.With(auth.RateLimit(30, 10)).Get("/search", searchH.Search)

		// Stats — feature-flag gated (premium or free based on flag, admins always pass)
		r.Group(func(r chi.Router) {
			r.Use(handler.RequireFeature(flagsRepo, "stats"))
			r.Get("/stats", statsH.Get)
			r.Get("/stats/summary", statsH.Summary)
		})

		// Trending — feature-flag gated
		r.Group(func(r chi.Router) {
			r.Use(handler.RequireFeature(flagsRepo, "trending"))
			r.Get("/trending/{category}", trendingH.Get)
		})

		// Portal reads — feature-flag gated
		r.Group(func(r chi.Router) {
			r.Use(handler.RequireFeature(flagsRepo, "portal"))
			r.Get("/portal", portalH.List)
			r.Get("/portal/status", portalH.Status)
		})

		// Rewatches (delete by rewatch ID)
		r.Delete("/rewatches/{id}", rewatchH.Delete)

		// Import
		r.Post("/import/mal", importH.ImportMAL)
		r.Post("/import/letterboxd", importH.ImportLetterboxd)
		r.Post("/import/goodreads", importH.ImportGoodreads)
		r.Get("/import/posters/missing-count", importH.MissingPosterCount) // legacy
		r.Post("/import/posters/refetch", importH.RefetchPosters)          // legacy
		r.Get("/import/metadata/missing-count", importH.MissingMetadataCount)
		r.Post("/import/metadata/refetch", importH.RefetchMetadata)

		// Admin
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin)
			// Portal writes (reads are flag-gated above)
			r.Post("/portal", portalH.Create)
			r.Put("/portal/reorder", portalH.Reorder)
			r.Patch("/portal/{id}", portalH.Update)
			r.Delete("/portal/{id}", portalH.Delete)
			r.Put("/portal/{id}/star", portalH.Star)
			r.Delete("/portal/{id}/star", portalH.Unstar)

			// Feature flags (superadmin check inside handler)
			r.Get("/admin/flags", flagsH.List)
			r.Patch("/admin/flags/{key}", flagsH.Set)

			r.Get("/admin/users", userH.AdminList)
			r.Get("/admin/users/{id}/stats", userH.AdminGetUserStats)
			r.Get("/admin/users/{id}/library", userH.AdminGetUserLibrary)
			r.Patch("/admin/users/{id}/flags", userH.AdminUpdateFlags)
			r.Delete("/admin/users/{id}", userH.AdminDeleteUser)
			r.Post("/admin/users/{id}/reset-password", userH.AdminResetPassword)
			r.Get("/admin/stats", userH.AdminStats)
			r.Get("/admin/health", healthH.ExternalServices)
			r.Get("/admin/invites", userH.AdminListInvites)
			r.Post("/admin/invites", userH.AdminCreateInvite)
			r.Delete("/admin/invites/{code}", userH.AdminDeleteInvite)
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

	// Background job: auto-move stale in_progress items to on_hold every 7 days
	go func() {
		ticker := time.NewTicker(7 * 24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := mediaRepo.AutoMarkInactive(context.Background()); err != nil {
				log.Printf("auto-inactive: %v", err)
			}
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
