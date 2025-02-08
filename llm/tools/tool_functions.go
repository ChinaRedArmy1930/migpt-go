package llm

import (
	"encoding/json"
	"time"
)

func getCurrentTime(json.RawMessage) string {
	return time.Now().Local().Format(time.DateTime)
}
