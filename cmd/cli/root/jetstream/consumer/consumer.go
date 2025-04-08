package consumer

import (
	clients_jetstream_consumer_add "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/consumer/add"
	clients_jetstream_consumer_delete "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/consumer/delete"
	clients_jetstream_consumer_info "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/consumer/info"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	cobra "github.com/spf13/cobra"
)

const use = "consumer"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	clients_jetstream_consumer_add.Init(command)
	clients_jetstream_consumer_info.Init(command)
	clients_jetstream_consumer_delete.Init(command)

	parentCmd.AddCommand(command)

}
