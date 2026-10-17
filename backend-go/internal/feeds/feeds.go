// Package feeds holds the curated source list and the fetch/clean logic for
// pulling security news out of each one.
package feeds

import (
	"context"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// Source is one RSS feed EagleInsight pulls from.
type Source struct {
	Name string
	URL  string
}

// Sources is the curated feed list. Add an entry here to pull from a new site.
var Sources = []Source{
	{"Krebs on Security", "https://krebsonsecurity.com/feed/"},
	{"The Hacker News", "https://feeds.feedburner.com/TheHackersNews"},
	{"BleepingComputer", "https://www.bleepingcomputer.com/feed/"},
	{"Dark Reading", "https://www.darkreading.com/rss.xml"},
	{"The Record", "https://therecord.media/feed/"},
	{"SecurityWeek", "https://www.securityweek.com/feed/"},
}

// Article is one story pulled from a feed, cleaned and ready to store.
type Article struct {
	Title     string
	Link      string
	Source    string
	Summary   string
	Published time.Time
}

const summaryMaxChars = 280

var (
	// WordPress feeds append "The post <title> appeared first on <site>." to every summary.
	wpFooter   = regexp.MustCompile(`(?s)\s*The post .* appeared first on .*$`)
	htmlTag    = regexp.MustCompile(`<[^>]*>`)
	whitespace = regexp.MustCompile(`\s+`)
)

// cleanSummary turns a feed's HTML/entity-laden text into short plain text.
// Feed content is untrusted, so tags are stripped here rather than relying on
// the client to escape them.
func cleanSummary(raw string) string {
	text := htmlTag.ReplaceAllString(raw, " ")
	text = html.UnescapeString(text)
	text = wpFooter.ReplaceAllString(text, "")
	text = strings.TrimSpace(whitespace.ReplaceAllString(text, " "))

	if len(text) > summaryMaxChars {
		cut := text[:summaryMaxChars]
		if i := strings.LastIndex(cut, " "); i > 0 {
			cut = cut[:i]
		}
		text = cut + "…"
	}
	return text
}

// FetchSource fetches and parses one feed. It never returns an error: a feed
// that is down or malformed just yields no articles, so one bad source can't
// break a refresh of everything else.
func FetchSource(ctx context.Context, s Source) []Article {
	fp := gofeed.NewParser()
	parsed, err := fp.ParseURLWithContext(s.URL, ctx)
	if err != nil {
		return nil
	}

	articles := make([]Article, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		title := cleanSummary(item.Title)
		link := strings.TrimSpace(item.Link)
		// Only web links: a feed must not be able to hand the client a javascript: URL.
		if title == "" || !(strings.HasPrefix(link, "https://") || strings.HasPrefix(link, "http://")) {
			continue
		}

		summary := item.Description
		if summary == "" {
			summary = item.Content
		}

		published := time.Now().UTC()
		if item.PublishedParsed != nil {
			published = item.PublishedParsed.UTC()
		} else if item.UpdatedParsed != nil {
			published = item.UpdatedParsed.UTC()
		}

		articles = append(articles, Article{
			Title:     title,
			Link:      link,
			Source:    s.Name,
			Summary:   cleanSummary(summary),
			Published: published,
		})
	}
	return articles
}

// FetchAll fetches every configured source and returns the combined list.
func FetchAll(ctx context.Context) []Article {
	var all []Article
	for _, s := range Sources {
		all = append(all, FetchSource(ctx, s)...)
	}
	return all
}

// Names returns just the configured source names, in order.
func Names() []string {
	names := make([]string, len(Sources))
	for i, s := range Sources {
		names[i] = s.Name
	}
	return names
}
