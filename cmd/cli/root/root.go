/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package root

import (
	"context"
	"os"

	callout "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/callout"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	clients "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/clients"
	handlers "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/handlers"
	jetstream "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream"

	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(cmd *cobra.Command) {
	cobra.CheckErr(cmd.Execute())
}

// ExecuteE adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteE(cmd *cobra.Command) error {
	return cmd.Execute()
}

// InitRootCmd initializes the root command
func InitRootCmd() *cobra.Command {
	// command represents the base command when called without any subcommands
	var command = &cobra.Command{
		Use:   "cli",
		Short: "A centrifugo client CLI tool",
		Long:  ``,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			logz := zerolog.New(os.Stdout).With().Caller().Timestamp().Logger()
			ctx = logz.WithContext(ctx)
			shared.SetContext(ctx)
			return nil
		},
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
	}
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kafkaCLI.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.

	callout.Init(command)
	handlers.Init(command)
	clients.Init(command)
	jetstream.Init(command)
	return command
}
