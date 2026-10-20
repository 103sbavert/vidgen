package config

import (
	"fmt"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validateOnce sync.Once
	validate     *validator.Validate
)

// getValidator initializes and initializes the singleton validator instance.
func getValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New()

		// Register custom validation for VideoResolution using RegexPatterns
		validate.RegisterValidation("resolution", func(fl validator.FieldLevel) bool {
			res, ok := fl.Field().Interface().(VideoResolution)
			if !ok {
				return false
			}
			return RegexPatterns.resolutionRegex.MatchString(string(res))
		})

		// Register custom validation for Color format
		validate.RegisterValidation("color", func(fl validator.FieldLevel) bool {
			var colorStr string
			val := fl.Field().Interface()

			switch c := val.(type) {
			case Color:
				colorStr = string(c)
			case string:
				colorStr = c
			default:
				return false
			}

			if _, exists := colorKeywords[colorStr]; exists {
				return true
			}

			return RegexPatterns.colorRegex.MatchString(colorStr)
		})

		// Register custom validation for VideoCodec name against validCodecs map
		validate.RegisterValidation("codec", func(fl validator.FieldLevel) bool {
			val := fl.Field().String()

			_, err := codecExtMap[val]
			return err
		})
	})
	return validate
}

// Validate reports whether the config can be built into a video using struct tags and custom rules.
func (c VideoConfig) Validate() error {
	v := getValidator()
	if err := v.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	return nil
}
