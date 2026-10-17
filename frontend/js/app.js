"use strict";

const PAGE_SIZE = 30;
const SAVED_KEY = "eagleinsight.saved";
const SAVED_FILTER = "__saved__";

const els = {
  articles: document.getElementById("articles"),
  filters: document.getElementById("filters"),
  search: document.getElementById("search"),
  status: document.getElementById("status"),
  loadMore: document.getElementById("load-more"),
  refresh: document.getElementById("refresh"),
};

const state = { filter: null, query: "", offset: 0 };

// Saved articles live in this browser only; each one is stored whole so the
// Saved view still works offline and after the article ages out of the feed.
function loadSaved() {
  try {
    return JSON.parse(localStorage.getItem(SAVED_KEY)) || {};
  } catch {
    return {};
  }
}

function storeSaved(saved) {
  try {
    localStorage.setItem(SAVED_KEY, JSON.stringify(saved));
  } catch {
    // Private browsing or a full quota: saving just doesn't persist.
  }
}

let saved = loadSaved();

function relativeTime(iso) {
  const seconds = Math.max(0, (Date.now() - new Date(iso).getTime()) / 1000);
  if (seconds < 3600) return `${Math.max(1, Math.round(seconds / 60))}m ago`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.round(seconds / 86400)}d ago`;
  return new Date(iso).toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function setStatus(message) {
  els.status.textContent = message || "";
  els.status.hidden = !message;
}

// Article fields come from third-party feeds, so they are only ever set with
// textContent, never innerHTML.
function renderArticle(article) {
  const item = document.createElement("li");
  item.className = "article";

  const meta = document.createElement("div");
  meta.className = "article-meta";

  const label = document.createElement("span");
  const source = document.createElement("span");
  source.className = "article-source";
  source.textContent = article.source;
  label.append(source, ` · ${relativeTime(article.published)}`);

  const save = document.createElement("button");
  save.className = "save-button";
  save.type = "button";
  const syncSave = () => {
    const isSaved = article.link in saved;
    save.textContent = isSaved ? "★" : "☆";
    save.setAttribute("aria-pressed", String(isSaved));
    save.setAttribute("aria-label", isSaved ? "Remove from saved" : "Save for later");
  };
  save.addEventListener("click", () => {
    if (article.link in saved) {
      delete saved[article.link];
    } else {
      saved[article.link] = article;
    }
    storeSaved(saved);
    if (state.filter === SAVED_FILTER) {
      showSaved();
    } else {
      syncSave();
    }
  });
  syncSave();
  meta.append(label, save);

  const title = document.createElement("a");
  title.className = "article-title";
  title.href = article.link;
  title.target = "_blank";
  title.rel = "noopener noreferrer";
  title.textContent = article.title;

  item.append(meta, title);

  if (article.summary) {
    const summary = document.createElement("p");
    summary.className = "article-summary";
    summary.textContent = article.summary;
    item.append(summary);
  }
  return item;
}

function showSaved() {
  const query = state.query.toLowerCase();
  const list = Object.values(saved)
    .filter((a) => !query || `${a.title} ${a.summary}`.toLowerCase().includes(query))
    .sort((a, b) => b.published.localeCompare(a.published));
  els.articles.replaceChildren(...list.map(renderArticle));
  els.loadMore.hidden = true;
  setStatus(list.length ? "" : "Nothing saved yet. Tap ☆ on an article to keep it here.");
}

async function loadArticles({ append = false } = {}) {
  if (state.filter === SAVED_FILTER) {
    showSaved();
    return;
  }
  if (!append) state.offset = 0;

  const params = new URLSearchParams({ limit: PAGE_SIZE, offset: state.offset });
  if (state.query) params.set("q", state.query);
  if (state.filter) params.set("source", state.filter);

  if (!append) setStatus("Loading…");
  let page;
  try {
    const response = await fetch(`/api/articles?${params}`);
    if (!response.ok) throw new Error(response.statusText);
    page = await response.json();
  } catch {
    setStatus("Couldn't load the feed. Check your connection and try again.");
    return;
  }

  const items = page.map(renderArticle);
  if (append) {
    els.articles.append(...items);
  } else {
    els.articles.replaceChildren(...items);
  }
  state.offset += page.length;
  els.loadMore.hidden = page.length < PAGE_SIZE;
  setStatus(els.articles.children.length ? "" : "No articles match.");
}

function renderFilters(sources) {
  const options = [
    { label: "All", value: null },
    { label: "★ Saved", value: SAVED_FILTER },
    ...sources.map((name) => ({ label: name, value: name })),
  ];
  const chips = options.map(({ label, value }) => {
    const chip = document.createElement("button");
    chip.className = "chip";
    chip.type = "button";
    chip.textContent = label;
    chip.setAttribute("aria-pressed", String(value === state.filter));
    chip.addEventListener("click", () => {
      state.filter = value;
      for (const other of els.filters.children) other.setAttribute("aria-pressed", "false");
      chip.setAttribute("aria-pressed", "true");
      window.scrollTo({ top: 0 });
      loadArticles();
    });
    return chip;
  });
  els.filters.replaceChildren(...chips);
}

let searchTimer;
els.search.addEventListener("input", () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => {
    state.query = els.search.value.trim();
    loadArticles();
  }, 250);
});

els.loadMore.addEventListener("click", () => loadArticles({ append: true }));

els.refresh.addEventListener("click", async () => {
  els.refresh.disabled = true;
  try {
    // A 429 just means the server refreshed recently; reloading the list is still right.
    await fetch("/api/refresh", { method: "POST" });
  } catch {
    // Offline: fall through and show whatever is cached.
  }
  await loadArticles();
  els.refresh.disabled = false;
});

async function init() {
  let sources = [];
  try {
    sources = await (await fetch("/api/sources")).json();
  } catch {
    // Offline with nothing cached: still show All and Saved.
  }
  renderFilters(sources);
  loadArticles();
}

if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register("service-worker.js").catch(() => {});
}

init();
