// Package session contains persistence adapters for the shared session manager.
package session

import (
	"database/sql"
	"time"
)

type SQLiteStore struct{ db *sql.DB }

func NewSQLiteStore(db *sql.DB) *SQLiteStore { return &SQLiteStore{db: db} }
func (s *SQLiteStore) Find(token string) ([]byte, bool, error) {
	var data []byte
	err := s.db.QueryRow(`SELECT data FROM sessions WHERE token=? AND expiry>?`, token, time.Now().UnixNano()).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return data, err == nil, err
}
func (s *SQLiteStore) Commit(token string, data []byte, expiry time.Time) error {
	_, err := s.db.Exec(`INSERT INTO sessions(token,data,expiry) VALUES(?,?,?) ON CONFLICT(token) DO UPDATE SET data=excluded.data,expiry=excluded.expiry`, token, data, expiry.UnixNano())
	return err
}
func (s *SQLiteStore) Delete(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
	return err
}
func (s *SQLiteStore) All() (map[string][]byte, error) {
	rows, err := s.db.Query(`SELECT token,data FROM sessions WHERE expiry>?`, time.Now().UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var token string
		var data []byte
		if err := rows.Scan(&token, &data); err != nil {
			return nil, err
		}
		out[token] = data
	}
	return out, rows.Err()
}
