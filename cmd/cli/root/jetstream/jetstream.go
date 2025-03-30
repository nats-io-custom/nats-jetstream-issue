package jetstream

import (
	clients_jetstream_consume "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/consume"
	clients_jetstream_consumer "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/consumer"
	clients_jetstream_create "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/create"
	clients_jetstream_info "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/info"
	clients_jetstream_kv "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/kv"
	clients_jetstream_publish "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/publish"
	clients_jetstream_publish_one "github.com/nats-io-custom/nats-jetstream-issue/cmd/cli/root/jetstream/publish_one"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	cobra "github.com/spf13/cobra"
)

const use = "jetstream"

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
	}

	clients_jetstream_create.Init(command)
	clients_jetstream_info.Init(command)
	clients_jetstream_consumer.Init(command)
	clients_jetstream_publish.Init(command)
	clients_jetstream_consume.Init(command)
	clients_jetstream_kv.Init(command)
	clients_jetstream_publish_one.Init(command)

	parentCmd.AddCommand(command)

}
