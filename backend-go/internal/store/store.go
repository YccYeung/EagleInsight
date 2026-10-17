// Package store handles SQLite storage for fetched articles.
package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver: no cgo, keeps the container build simple

	"eagleinsight/internal/feeds"
)

const schema = `
CREATE TABLE IF NOT EXISTS articles (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	title      TEXT NOT NULL,
	link       TEXT NOT NULL UNIQUE,
	source     TEXT NOT NULL,
	summary    TEXT,
	published  TEXT NOT NULL,
	fetched_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_articles_source    ON articles(source);
CREATE INDEX IF NOT EXISTS idx_articles_published ON articles(published);
`

// Article is a stored article as returned to API callers.
type Article struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Source    string `json:"source"`
	Summary   string `json:"summary"`
	Published string `json:"published"`
	FetchedAt string `json:"fetched_at"`
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite only allows one writer at a time; a single connection avoids
	// "database is locked" errors under concurrent requests.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// UpsertArticles inserts new articles, skipping ones already stored (same
// link). Returns how many were newly inserted.
func (s *Store) UpsertArticles(articles []feeds.Article) (int, error) {
	inserted := 0
	for _, a := range articles {
		res, err := s.db.Exec(
			`INSERT OR IGNORE INTO articles (title, link, source, summary, published) VALUES (?, ?, ?, ?, ?)`,
			a.Title, a.Link, a.Source, a.Summary, a.Published.Format(time.RFC3339),
		)
		if err != nil {
			return inserted, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			inserted++
		}
	}
	return inserted, nil
}

type Query struct {
	Search string
	Source string
	Limit  int
	Offset int
}

func (s *Store) QueryArticles(q Query) ([]Article, error) {
	sqlStr := "SELECT id, title, link, source, summary, published, fetched_at FROM articles WHERE 1=1"
	var args []any

	if q.Search != "" {
		sqlStr += " AND (title LIKE ? OR summary LIKE ?)"
		like := "%" + q.Search + "%"
		args = append(args, like, like)
	}
	if q.Source != "" {
		sqlStr += " AND source = ?"
		args = append(args, q.Source)
	}
	sqlStr += " ORDER BY published DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Article
	for rows.Next() {
		var a Article
		if err := rows.Scan(&a.ID, &a.Title, &a.Link, &a.Source, &a.Summary, &a.Published, &a.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) ArticleCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
	return count, err
}
