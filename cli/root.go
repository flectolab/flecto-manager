package cli

import (
	"fmt"
	"log/slog"
	"path"
	"path/filepath"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/crypt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	ConfigName = "manager"
	LogLevel   = "level"
	Name       = "flecto-manager"
)

func GetDefaultConfigPath() string {
	return filepath.Join("/etc", commonTypes.Namespace)
}

func GetRootCmd(ctx *context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:               Name,
		Short:             fmt.Sprintf("%s: Dynamic HTTP redirect and traffic routing management for modern infrastructure.", Name),
		PersistentPreRunE: GetRootPreRunEFn(ctx, true),
	}

	cmd.PersistentFlags().StringP(ConfigName, "c", "", "Define config path")
	cmd.PersistentFlags().StringP(LogLevel, "l", "INFO", "Define log level")
	_ = viper.BindPFlag(ConfigName, cmd.Flags().Lookup(ConfigName))
	_ = viper.BindPFlag(LogLevel, cmd.Flags().Lookup(LogLevel))

	cmd.AddCommand(
		GetStartCmd(ctx),
		GetDBCmd(ctx),
		GetUserCmd(ctx),
		GetSecurityCmd(ctx),
		GetVersionCmd(),
		GetValidateCmd(ctx),
	)

	return cmd
}

func GetRootPreRunEFn(ctx *context.Context, validateCfg bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		var err error
		initConfig(ctx, cmd)

		if errValidate := validateConfig(ctx); validateCfg && errValidate != nil {
			return errValidate
		}

		if err = setupCrypt(ctx); err != nil {
			return err
		}

		logLevelFlagStr, _ := cmd.Flags().GetString(LogLevel)
		if logLevelFlagStr != "" {
			level := slog.LevelInfo
			err = level.UnmarshalText([]byte(logLevelFlagStr))
			if err != nil {
				return err
			}
			ctx.LogLevel.Set(level)
		}
		return nil
	}
}

// setupCrypt builds the cipher from the configuration and wires it to GORM. A
// missing key is not an error.
func setupCrypt(ctx *context.Context) error {
	cipher, err := crypt.NewFromKey(ctx.Config.Security.EncryptionKey)
	if err != nil {
		return err
	}

	ctx.Cipher = cipher
	crypt.RegisterSerializer(cipher)

	if crypt.IsUnavailable(cipher) {
		ctx.Logger.Warn("no security.encryption_key configured, features storing secrets are disabled")
	}
	return nil
}

func initConfig(ctx *context.Context, cmd *cobra.Command) {

	viper.AddConfigPath(GetDefaultConfigPath())
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvPrefix(Name)
	viper.SetConfigName(ConfigName)
	viper.SetConfigType("yaml")

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		panic(err)
	}

	configPath := viper.GetString(ConfigName)

	if configPath != "" {
		viper.SetConfigFile(configPath)
		configDir := path.Dir(configPath)
		if configDir != "." {
			viper.AddConfigPath(configDir)
		}
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Println(err)
	}

	err := viper.Unmarshal(ctx.Config)
	if err != nil {
		panic(fmt.Errorf("unable to decode into config struct, %v", err))
	}

}
