package config

import (
	"encoding/json"
	"strings"
)

func (r *VideoResolution) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	*r = VideoResolution(s)
	return nil
}

func (c *Color) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.TrimPrefix(s, "#")
	s = strings.ToLower(s)

	color, ok := colorKeywords[s]
	if ok {
		s = color
	}

	*c = Color(s)
	return nil
}
