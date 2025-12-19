package models

import "time"

type User struct {
	ID             int        `json:"id"`
	Username       string     `json:"username"`
	Email          *string    `json:"email"`
	PasswordHash   string     `json:"-"`
	DisplayName    *string    `json:"display_name"`
	AvatarURL      *string    `json:"avatar_url"`
	IsAdmin        bool       `json:"is_admin"`
	IsPremium      bool       `json:"is_premium"`
	IsPublic       bool       `json:"is_public"`
	FailedAttempts int        `json:"-"`
	LockedUntil    *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type MediaType   string
type MediaStatus string

const (
	MediaTypeFilm   MediaType = "film"
	MediaTypeTVShow MediaType = "tv_show"
	MediaTypeBook   MediaType = "book"
	MediaTypeAnime  MediaType = "anime"

	StatusWantTo     MediaStatus = "want_to"
	StatusInProgress MediaStatus = "in_progress"
	StatusCompleted  MediaStatus = "completed"
	StatusDropped    MediaStatus = "dropped"
	StatusOnHold     MediaStatus = "on_hold"
)

type MediaItem struct {
	ID          int            `json:"id"`
	UserID      int            `json:"user_id"`
	MediaType   MediaType      `json:"media_type"`
	ExternalID  *string        `json:"external_id"`
	Title       string         `json:"title"`
	Year        *int           `json:"year"`
	PosterURL   *string        `json:"poster_url"`
	Metadata    map[string]any `json:"metadata"`
	Status      MediaStatus    `json:"status"`
	Rating      *float64       `json:"rating"`
	ReviewText  *string        `json:"review_text"`
	StartedAt   *time.Time     `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type TvEpisodeLog struct {
	ID            int        `json:"id"`
	MediaItemID   int        `json:"media_item_id"`
	SeasonNumber  int        `json:"season_number"`
	EpisodeNumber int        `json:"episode_number"`
	WatchedAt     *time.Time `json:"watched_at"`
	Rating        *float64   `json:"rating"`
	Note          *string    `json:"note"`
}

type ChapterStatus string

const (
	ChapterUnread     ChapterStatus = "unread"
	ChapterInProgress ChapterStatus = "in_progress"
	ChapterCompleted  ChapterStatus = "completed"
)

type BookChapterLog struct {
	ID            int           `json:"id"`
	MediaItemID   int           `json:"media_item_id"`
	ChapterNumber int           `json:"chapter_number"`
	ChapterTitle  *string       `json:"chapter_title"`
	StartPage     *int          `json:"start_page"`
	EndPage       *int          `json:"end_page"`
	Status        ChapterStatus `json:"status"`
	Note          *string       `json:"note"`
	StartedAt     *time.Time    `json:"started_at"`
	CompletedAt   *time.Time    `json:"completed_at"`
}

type UserList struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	IsPublic    bool       `json:"is_public"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Items       []ListItem `json:"items"`
}

type ListItem struct {
	ID          int        `json:"id"`
	ListID      int        `json:"list_id"`
	MediaItemID int        `json:"media_item_id"`
	Position    int        `json:"position"`
	AddedAt     time.Time  `json:"added_at"`
	MediaItem   *MediaItem `json:"media_item,omitempty"`
}

type SearchResult struct {
	Source      string         `json:"source"`
	MediaType   string         `json:"media_type"`
	ExternalID  string         `json:"external_id"`
	Title       string         `json:"title"`
	Year        *int           `json:"year"`
	PosterURL   *string        `json:"poster_url"`
	Description *string        `json:"description"`
	Extra       map[string]any `json:"extra"`
}

type PaginatedMedia struct {
	Items   []MediaItem `json:"items"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Pages   int         `json:"pages"`
}
