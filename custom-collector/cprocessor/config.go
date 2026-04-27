package cprocessor

import "errors"

type Config struct {
	Environment     string `mapstructure:"environment"`
	AddLogAttribute bool   `mapstructure:"add_log_attribute"`
	RedactUserEmail bool   `mapstructure:"redact_user_email"`
}

func (c *Config) Validate() error {
	if c.Environment == "" {
		return errors.New("environment must not be empty")
	}

	return nil
}
