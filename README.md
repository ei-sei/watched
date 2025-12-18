# Watched

![Architecture.png](architecture.png)


```bash
media-tracker/
├── frontend/          # React + Tailwind
│   ├── src/
│   │   ├── components/
│   │   ├── pages/     # Home, Search, MediaDetail, Progress
│   │   └── api/       # TMDB + backend API calls
│   └── Dockerfile
├── backend/           # FastAPI
│   ├── app/
│   │   ├── routers/   # media.py, reviews.py, auth.py, search.py
│   │   ├── models/    # SQLAlchemy models
│   │   └── services/  # tmdb.py, openlibrary.py
│   └── Dockerfile
├── nginx/
│   └── nginx.conf
└── docker-compose.yml
```

