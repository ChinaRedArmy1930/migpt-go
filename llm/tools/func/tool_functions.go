package llm

import (
	"encoding/json"
	"time"
)

//tool:register
func GetCurrentTime(json.RawMessage) (string, error) {
	return time.Now().Local().Format(time.DateTime), nil
}
