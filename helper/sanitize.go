package helper

import (
	"fmt"

	"cosmossdk.io/log"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func SanitizeConfig(rootViper *viper.Viper, logger log.Logger) error {
	if rootViper == nil {
		return fmt.Errorf("invalid config: viper instance is nil")
	}

	var appCfg CustomAppConfig
	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)
	if err := rootViper.Unmarshal(&appCfg, viper.DecodeHook(decodeHook)); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if notes, kv := appCfg.Sanitize(); len(notes) > 0 {
		logger.Warn("Detected some configuration values to be sanitized")
		for _, n := range notes {
			logger.Warn("Sanitizing configs: adjusted:", n)
		}
		// write sanitized values back into Viper
		for k, v := range kv {
			rootViper.Set(k, v)
		}
	}
	return nil
}
