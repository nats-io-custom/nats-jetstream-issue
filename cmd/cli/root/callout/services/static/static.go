package static

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"

	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	status "github.com/gogo/status"
	contracts_users "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/users"
	services_user_store_inmemory "github.com/nats-io-custom/nats-jetstream-issue/internal/services/user_store/inmemory"
	jwt "github.com/nats-io/jwt/v2"
	nkeys "github.com/nats-io/nkeys"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
	callout "github.com/synadia-io/callout.go"
	codes "google.golang.org/grpc/codes"
)

const use = "static"

var (
	appInputs        = shared.NewInputs()
	usersFile string = "./configs/users.json"
)
var wellknownAccounts = map[string]string{
	"svc": "SVC",
	"sys": "SYS",
}

// Init command
func Init(parentCmd *cobra.Command) {
	var command = &cobra.Command{
		Use:               use,
		Short:             use,
		PersistentPreRunE: cobra_utils.ParentPersistentPreRunE,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := shared.GetContext()
			log := zerolog.Ctx(ctx).With().Str("command", use).Logger()

			printer := cobra_utils.NewPrinter()
			printer.EnableColors = true
			printer.PrintBold(cobra_utils.Bold, use)

			builder := di.Builder()
			di.AddInstance[*contracts_users.UserStoreConfig](builder,
				&contracts_users.UserStoreConfig{
					UserFile: usersFile,
				})
			services_user_store_inmemory.AddSingletonUserStore(builder)
			ctn := builder.Build()

			userStore, err := di.TryGet[contracts_users.IUserStore](ctn)
			if err != nil {
				log.Error().Err(err).Msg("failed to get user store")
				return err
			}

			nc, err := appInputs.MakeConn(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to connect to nats server")
				return err
			}
			defer nc.Drain()
			printer.Infof("%s connected to %s", appInputs.NatsUser, nc.ConnectedUrl())

			// parse the private key
			akp, err := nkeys.FromSeed([]byte(appInputs.IssuerSeed))
			if err != nil {
				log.Error().Err(err).Msg("error parsing issuer seed")
				return err
			}
			akpPublickKey, _ := akp.PublicKey()
			log.Info().Str("issuer", akpPublickKey).Msg("issuer")

			if !shared.FileExists(usersFile) {
				log.Error().Str("file", usersFile).Msg("file does not exist")
				return status.Error(codes.Internal, "file does not exist")
			}
			getUsersResponse, err := userStore.GetUsers(ctx)
			if err != nil {
				log.Error().Err(err).Msg("error getting users")
				return err
			}
			users := getUsersResponse.Users
			printer.Println(cobra_utils.Blue, fluffycore_utils.PrettyJSON(users))
			// Parse the xkey seed if present.
			var curveKeyPair nkeys.KeyPair
			if fluffycore_utils.IsNotEmptyOrNil(appInputs.XKeySeed) {
				curveKeyPair, err = nkeys.FromSeed([]byte(appInputs.XKeySeed))
				if err != nil {
					log.Error().Err(err).Msg("error parsing xkey seed")
					return status.Error(codes.Internal, "error parsing xkey seed")
				}
			}
			if curveKeyPair != nil {
				curveKeyPairPublicKey, _ := curveKeyPair.PublicKey()
				log.Info().Str("xkey", curveKeyPairPublicKey).Msg("xkey")
			}
			// a function that creates the users
			authorizer := func(req *jwt.AuthorizationRequest) (string, error) {
				// peek at the req for information - for brevity
				// in the example, we simply allow them in
				log.Info().Str("user", req.UserNkey).Msg("authorizing")

				userParts := strings.Split(req.ConnectOptions.Username, "@")
				if len(userParts) != 2 {
					log.Error().Str("user", req.UserNkey).Msg("invalid user format")
					return "", status.Error(codes.PermissionDenied, "invalid user format")
				}
				username := userParts[0]
				accountWanted := strings.ToLower(userParts[1])
				password := req.ConnectOptions.Password

				printer.Printf(cobra_utils.Blue, "username: %s, password: %s\n", username, password)

				authenticateUserResponse, err := userStore.AuthenticateUser(ctx,
					&contracts_users.AuthenticateUserRequest{
						UserName: username,
						Password: password,
						Account:  accountWanted,
					})
				if err != nil {
					log.Error().Err(err).Msg("error authenticating user")
					return "", err
				}

				user := authenticateUserResponse.User
				audience, ok := wellknownAccounts[accountWanted]
				if !ok {
					printer.Printf(cobra_utils.Red, "UNAUTHORIZED: invalid user account: %s\n", username)
					return "", status.Error(codes.PermissionDenied, "invalid user account")
				}
				printer.Println(cobra_utils.Blue, fluffycore_utils.PrettyJSON(user))

				// use the server specified user nkey
				uc := jwt.NewUserClaims(req.UserNkey)
				// put the user in the global account
				uc.Audience = audience
				// add whatever permissions you need
				uc.Sub.Allow.Add(user.Sub.Allow...)
				uc.Pub.Allow.Add(user.Pub.Allow...)

				uc.Sub.Deny.Add(user.Sub.Deny...)
				uc.Pub.Deny.Add(user.Pub.Deny...)

				// perhaps add an expiration to the JWT
				uc.Expires = time.Now().Unix() + 90
				return uc.Encode(akp)
			}
			// start the service
			_, err = callout.NewAuthorizationService(nc, callout.Authorizer(authorizer), callout.ResponseSignerKey(akp))
			if err != nil {
				log.Error().Err(err).Msg("error starting service")
				return err
			}
			// don't exit until sigterm
			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
			<-quit
			return nil
		},
	}
	appInputs.NatsUser = "auth"
	appInputs.NatsPass = "auth"
	appInputs.IssuerSeed = "SAAEXFSYMLINXLKR2TG5FLHCJHLU62B3SK3ESZLGP4B4XGLUNXICW3LGAY"

	shared.InitCommonConnFlags(appInputs, command)

	flagName := "issuer.seed"
	defaultS := appInputs.IssuerSeed
	command.Flags().StringVar(&appInputs.IssuerSeed, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "users.file"
	defaultS = usersFile
	command.Flags().StringVar(&usersFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}
