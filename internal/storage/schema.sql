PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
CREATE TABLE IF NOT EXISTS settings (id INTEGER PRIMARY KEY CHECK(id=1), json TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS profiles (id TEXT PRIMARY KEY, json TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS sessions (
  id TEXT PRIMARY KEY, profile TEXT NOT NULL, started_at TEXT NOT NULL,
  ended_at TEXT, seconds INTEGER NOT NULL DEFAULT 0,
  outcome TEXT NOT NULL DEFAULT 'active', app_blocks INTEGER NOT NULL DEFAULT 0,
  site_blocks INTEGER NOT NULL DEFAULT 0, last_seen TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_started ON sessions(started_at);
PRAGMA user_version = 1;
