# EagleInsight

Security news from several sources in one feed, built as an installable web app (PWA) you can read on your phone.

## What it does

- Pulls articles from Krebs on Security, The Hacker News, BleepingComputer, Dark Reading, The Record, and SecurityWeek.
- Shows them newest first, with search and a filter per source.
- Lets you save articles to read later. Saved articles are kept in the browser.
- Refreshes the feeds every 3 hours, or when you tap the refresh button.
- Keeps the last feed you loaded available offline.

## How it works

```
RSS feeds ──> backend (FastAPI) ──> SQLite ──> /api/articles ──> frontend (PWA)
```

- `backend/app/feeds.py`: the source list, fetching, and cleaning. Feed content is untrusted, so HTML is stripped and only `http(s)` links are kept.
- `backend/app/db.py`: SQLite storage. Articles are de-duplicated by link.
- `backend/app/main.py`: the API, the refresh schedule, and serving the frontend.
- `frontend/`: plain HTML, CSS, and JavaScript with no build step, plus the manifest and service worker that make it installable.

## Run it locally

You need [uv](https://docs.astral.sh/uv/).

```sh
cd backend
uv run uvicorn app.main:app --port 8077
```

Then open http://127.0.0.1:8077. The first start takes a few seconds while it fetches the feeds.

## API

| Endpoint | What it returns |
|---|---|
| `GET /api/articles?q=&source=&limit=&offset=` | Articles, newest first |
| `GET /api/sources` | The configured source names |
| `POST /api/refresh` | Fetches the feeds now (at most once a minute) |
| `GET /api/health` | Status and article count |

## Adding a source

Add a `(name, feed URL)` pair to `SOURCES` in `backend/app/feeds.py`.

## Roadmap

- Deploy behind HTTPS so it can be installed on a phone
- Tests and CI
- CVE and CISA KEV data alongside the news
- Push notifications for major stories
