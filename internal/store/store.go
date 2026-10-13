// Package store는 감시 대상의 마지막 값과 변경 이력을 SQLite에 저장한다.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS target_state (
	target_name   TEXT PRIMARY KEY,
	last_value    TEXT NOT NULL,
	last_checked_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS change_history (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	target_name TEXT NOT NULL,
	old_value   TEXT,
	new_value   TEXT NOT NULL,
	changed_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_change_history_target
	ON change_history (target_name, changed_at);
`

// Store는 SQLite 연결을 감싼다.
type Store struct {
	db *sql.DB
}

// Open은 path의 SQLite 파일을 열고(없으면 생성) 스키마를 적용한다.
// path에 ":memory:"를 주면 테스트용 인메모리 DB로 동작한다.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite 연결 실패 (%s): %w", path, err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("스키마 적용 실패: %w", err)
	}

	return &Store{db: db}, nil
}

// Close는 DB 연결을 닫는다.
func (s *Store) Close() error {
	return s.db.Close()
}

// TargetState는 감시 대상의 마지막으로 확인된 값이다.
type TargetState struct {
	TargetName    string
	LastValue     string
	LastCheckedAt time.Time
}

// GetState는 target의 마지막 상태를 반환한다. 아직 한 번도 확인하지 않았으면
// (nil, nil)을 반환한다.
func (s *Store) GetState(targetName string) (*TargetState, error) {
	row := s.db.QueryRow(
		`SELECT target_name, last_value, last_checked_at FROM target_state WHERE target_name = ?`,
		targetName,
	)

	var state TargetState
	var checkedAt string
	err := row.Scan(&state.TargetName, &state.LastValue, &checkedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("상태 조회 실패 (%s): %w", targetName, err)
	}

	state.LastCheckedAt, err = time.Parse(time.RFC3339, checkedAt)
	if err != nil {
		return nil, fmt.Errorf("저장된 시각 파싱 실패 (%s): %w", targetName, err)
	}
	return &state, nil
}

// SetState는 target의 현재 값을 upsert한다.
func (s *Store) SetState(targetName, value string, checkedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO target_state (target_name, last_value, last_checked_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(target_name) DO UPDATE SET
			last_value = excluded.last_value,
			last_checked_at = excluded.last_checked_at`,
		targetName, value, checkedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("상태 저장 실패 (%s): %w", targetName, err)
	}
	return nil
}

// RecordChange는 변경 이력을 한 건 추가한다.
func (s *Store) RecordChange(targetName, oldValue, newValue string, changedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO change_history (target_name, old_value, new_value, changed_at)
		 VALUES (?, ?, ?, ?)`,
		targetName, oldValue, newValue, changedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("변경 이력 저장 실패 (%s): %w", targetName, err)
	}
	return nil
}

// ChangeRecord는 change_history 한 행이다.
type ChangeRecord struct {
	TargetName string
	OldValue   string
	NewValue   string
	ChangedAt  time.Time
}

// History는 target의 변경 이력을 최신순으로 반환한다.
func (s *Store) History(targetName string) ([]ChangeRecord, error) {
	rows, err := s.db.Query(
		`SELECT target_name, old_value, new_value, changed_at
		 FROM change_history
		 WHERE target_name = ?
		 ORDER BY changed_at DESC`,
		targetName,
	)
	if err != nil {
		return nil, fmt.Errorf("변경 이력 조회 실패 (%s): %w", targetName, err)
	}
	defer rows.Close()

	var records []ChangeRecord
	for rows.Next() {
		var r ChangeRecord
		var changedAt string
		if err := rows.Scan(&r.TargetName, &r.OldValue, &r.NewValue, &changedAt); err != nil {
			return nil, fmt.Errorf("변경 이력 읽기 실패 (%s): %w", targetName, err)
		}
		r.ChangedAt, err = time.Parse(time.RFC3339, changedAt)
		if err != nil {
			return nil, fmt.Errorf("저장된 시각 파싱 실패 (%s): %w", targetName, err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
