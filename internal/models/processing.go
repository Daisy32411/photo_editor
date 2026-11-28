package models

import "encoding/json"

type Filter struct {
	Name       string          `json:"name"`
	Parameters json.RawMessage `json:"parameters,omitempty"`
}

type ImageTaskMessage struct {
	TaskID string `json:"task_id"`
	Image  string `json:"image"`
	Filter Filter `json:"filter"`
}
