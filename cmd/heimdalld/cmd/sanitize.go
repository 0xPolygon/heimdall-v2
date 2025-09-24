package heimdalld

import (
	"fmt"

	"github.com/0xPolygon/heimdall-v2/helper"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func SanitizeConfig(rootViper *viper.Viper) error {
	var appCfg helper.CustomAppConfig
	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)
	if err := rootViper.Unmarshal(&appCfg, viper.DecodeHook(decodeHook)); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	if notes, kv := appCfg.Sanitize(); len(notes) > 0 {
		fmt.Println("## Detected some configuration values to be sanitized")
		for _, n := range notes {
			fmt.Println("[config] adjusted:", n)
		}
		// write sanitized values back into Viper
		for k, v := range kv {
			rootViper.Set(k, v)
		}
	}
	return nil
}
