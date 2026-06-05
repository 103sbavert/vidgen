package configuration

import (
	"encoding/json"
	"fmt"
	"strings"
)

func (c *VideoCodec) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	codec, ok := validCodecs[s]
	if !ok {
		return fmt.Errorf("unknown codec: %s", s)
	}
	*c = codec
	return nil
}

func (r *VideoResolution) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	if !RegexPatterns.resolutionRegex.MatchString(s) {
		return fmt.Errorf("invalid resolution: %s", s)
	}

	*r = VideoResolution(s)
	return nil
}

func (r *Color) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s = strings.TrimPrefix(s, "#")
	s = strings.ToLower(s)

	if colorKeywords[s] != "" {
		s = colorKeywords[s]
	} else if !RegexPatterns.colorRegex.MatchString(s) {
		return fmt.Errorf("invalid color: %s", s)
	}

	*r = Color(s)
	return nil
}
