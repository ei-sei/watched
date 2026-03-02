# Watched

A self-hosted personal media tracker for movies, TV shows, anime, and books. Built as a PWA with full offline support, so it works seamlessly on both desktop and mobile.

Live at [brsti.uk](https://brsti.uk) — invite-only registration.

---

## Features

### Media Tracking
- Track **movies, TV shows, anime, and books** in a single library
- Status tracking: Want to Watch / In Progress / Completed / Dropped / On Hold
- Star ratings (1–10), written reviews, and start/finish dates
- Rewatch logging for films, TV shows, and anime

### Episode & Chapter Tracking
- Per-season episode progress for TV shows and anime
- Tap `+`/`−` to log individual episodes, or type a number to jump to any episode count
- Visual progress bars with per-episode timestamp dots (shows when each episode was watched)
- Chapter-by-chapter tracking for books

### Search
- Unified search across all media types from a single input
- Sources: TMDB (movies & TV), Jikan/AniList (anime), Google Books + OpenLibrary (books)
- Tap any result to open a detail sheet with full synopsis and one-tap add to library
- Filters by type (Movies, TV, Books, Anime) with correct source routing

### Trending
- Trending and seasonal charts for anime, movies, and TV shows
- Sourced from AniList (anime) and TMDB (movies/TV)
- Tap any poster to see a synopsis sheet and add directly to your library

### Stats
- Watch time estimates, completion rates, ratings breakdown
- Per-media-type summaries
- Premium-gated feature

### Public Profiles & Lists
- Optional public profile page at `/u/:username`
- Create and share curated lists with a public link

### PWA
- Installable on Android and iOS as a home screen app
- Full offline support — all library reads come from local IndexedDB (Dexie), never the network
- Session persistence across app restarts on Android (refresh token stored locally)
- Bottom navigation optimised for mobile thumb reach

---

## Stack

| Layer | Technology |
|---|---|
| Backend | Go (`chi`, `pgx`, `golang-migrate`) |
| Frontend | React + TypeScript, Vite, Tailwind CSS |
| State | TanStack Query v5 (server), Dexie/IndexedDB (local) |
| Database | PostgreSQL 16 |
| Auth | JWT + refresh tokens, invite-only registration |
| Infra | Docker Compose, GitHub Actions CI/CD, GHCR |
| HTTPS | Let's Encrypt + certbot |

---

## External APIs

| API | Used for |
|---|---|
| TMDB | Movie & TV metadata, search, trending |
| Jikan (MAL) | Anime metadata, multi-season grouping |
| AniList | Anime search, trending |
| Google Books | Book metadata and search |
| OpenLibrary | Book search fallback |

All external API calls go through the backend — never from the browser.

---

## Self-Hosting

### Prerequisites
- Docker + Docker Compose
- A domain with DNS pointing to your server
- Certbot for HTTPS

### Quick Start

```bash
git clone https://github.com/ei-sei/watched.git
cd watched
```

Set environment variables in a `.env` file:

```env
POSTGRES_PASSWORD=yourpassword
JWT_SECRET=a-long-random-secret
TMDB_API_KEY=your-tmdb-key
GOOGLE_BOOKS_KEY=your-google-books-key   # optional
DB_SIZE_LIMIT_BYTES=1073741824           # 1 GB default
```

```bash
docker compose -f docker-compose.selfhost.yml up -d
```

The backend runs database migrations automatically on startup.

### Creating the First User

Registration requires an invite code. Create one directly in the database:

```sql
INSERT INTO invite_codes (code) VALUES ('YOURCODE');
```

Then register at `https://yourdomain/register?invite=YOURCODE`.

Promote your account to admin:

```sql
UPDATE users SET is_admin = true, is_premium = true WHERE username = 'yourusername';
```

---

## Development

```bash
# Backend
cd backend
go build ./...
go vet ./...

# Frontend
cd frontend
npm install
npm run dev       # dev server at localhost:5173
npm run build     # production build (includes type-check)
npm run lint
```

### Database Migrations

Migrations live in `backend/migrations/` as paired `.up.sql` / `.down.sql` files. Never edit existing migrations — always add a new numbered pair.

---

## Project Structure

```
watched/
├── backend/
│   ├── cmd/api/         # Entry point, routing
│   ├── internal/
│   │   ├── auth/        # JWT, password hashing
│   │   ├── config/      # Environment config
│   │   ├── handler/     # HTTP handlers
│   │   ├── models/      # Shared types
│   │   └── repository/  # Raw SQL queries (pgx, no ORM)
│   └── migrations/      # Paired .up/.down SQL migrations
└── frontend/
    └── src/
        ├── api/          # Typed API clients
        ├── components/   # UI + tracking components
        ├── hooks/        # TanStack Query + Dexie hooks
        ├── offline/      # Dexie schema, sync queue
        ├── pages/        # Route-level page components
        └── types/        # Shared TypeScript types
```
