package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type StringList []string

func (s *StringList) Scan(value any) error {
	if value == nil {
		*s = nil
		return nil
	}

	var data []byte

	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return fmt.Errorf("unsupported StringList value: %T", value)
	}

	return json.Unmarshal(data, s)
}

func (s StringList) Value() (driver.Value, error) {
	return json.Marshal(s)
}

type JobStatusUpdate struct {
	ID           int64      `db:"id"`
	Status       string     `db:"status"`
	Retries      *int       `db:"retries"`
	Progress     *string    `db:"progress"`
	ErrorMessage *string    `db:"error_message"`
	StartedAt    *time.Time `db:"started_at"`
	FinishedAt   *time.Time `db:"finished_at"`
}
