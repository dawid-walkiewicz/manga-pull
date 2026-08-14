package db

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
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
