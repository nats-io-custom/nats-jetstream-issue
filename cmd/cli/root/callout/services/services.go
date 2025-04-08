package services

import (
	callout_services_operator_mode_url_resolver "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/callout/services/operator_mode_url_resolver"
	callout_services_static "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/callout/services/static"
	callout_services_url_resolver "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/callout/services/url_resolver"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	cobra "github.com/spf13/cobra"
)

const use = "services"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	callout_services_static.Init(command)
	callout_services_url_resolver.Init(command)
	callout_services_operator_mode_url_resolver.Init(command)

	parentCmd.AddCommand(command)

}
