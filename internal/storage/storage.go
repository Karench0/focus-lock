package storage

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/Karench0/focus-lock/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var Schema string

type Store struct{ db *sql.DB }

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err = db.Exec(Schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	// Never restore enforcement after a restart. Count only persisted heartbeats.
	_, err = db.Exec(`UPDATE sessions SET outcome='recovered',ended_at=last_seen WHERE outcome='active'`)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Settings() (model.Settings, error) {
	v := model.Settings{Language: "ru", ExitDelaySeconds: 60}
	var data string
	err := s.db.QueryRow("SELECT json FROM settings WHERE id=1").Scan(&data)
	if err == sql.ErrNoRows {
		return v, nil
	}
	if err != nil {
		return v, err
	}
	err = json.Unmarshal([]byte(data), &v)
	return v, err
}
func (s *Store) SaveSettings(v model.Settings) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO settings VALUES(1,?) ON CONFLICT(id) DO UPDATE SET json=excluded.json", string(b))
	return err
}
func (s *Store) Profiles() ([]model.Profile, error) {
	rows, err := s.db.Query("SELECT json FROM profiles ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Profile{}
	for rows.Next() {
		var b string
		var p model.Profile
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(b), &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Store) SaveProfile(p model.Profile) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = s.db.Exec("INSERT INTO profiles VALUES(?,?) ON CONFLICT(id) DO UPDATE SET json=excluded.json", p.ID, string(b))
	return err
}
func (s *Store) DeleteProfile(id string) error {
	_, err := s.db.Exec("DELETE FROM profiles WHERE id=?", id)
	return err
}
func (s *Store) Start(id, profile string, at time.Time) error {
	_, err := s.db.Exec("INSERT INTO sessions(id,profile,started_at,last_seen) VALUES(?,?,?,?)", id, profile, at.Format(time.RFC3339), at.Format(time.RFC3339))
	return err
}
func (s *Store) Heartbeat(id string, seconds, app, site int) error {
	_, err := s.db.Exec("UPDATE sessions SET last_seen=?,seconds=?,app_blocks=?,site_blocks=? WHERE id=? AND outcome='active'", time.Now().Format(time.RFC3339), seconds, app, site, id)
	return err
}
func (s *Store) Finish(r model.Record) error {
	_, err := s.db.Exec("UPDATE sessions SET ended_at=?,seconds=?,outcome=?,app_blocks=?,site_blocks=?,last_seen=? WHERE id=? AND outcome='active'", r.EndedAt, r.Seconds, r.Outcome, r.AppBlocks, r.SiteBlocks, r.EndedAt, r.ID)
	return err
}
func (s *Store) Records() ([]model.Record, error) {
	rows, err := s.db.Query("SELECT id,profile,started_at,COALESCE(ended_at,''),seconds,outcome,app_blocks,site_blocks FROM sessions ORDER BY started_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Record{}
	for rows.Next() {
		var r model.Record
		if err = rows.Scan(&r.ID, &r.Profile, &r.StartedAt, &r.EndedAt, &r.Seconds, &r.Outcome, &r.AppBlocks, &r.SiteBlocks); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
