package operator_mode_url_resolver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"
	contracts_users "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/users"
	services_user_strore_inmemory "github.com/nats-io-custom/nats-jetstream-issue/internal/services/user_store/inmemory"
	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"
	jwt "github.com/nats-io/jwt/v2"
	nats "github.com/nats-io/nats.go"
	nkeys "github.com/nats-io/nkeys"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
	callout "github.com/synadia-io/callout.go"
)

const use = "operator_mode_url_resolver"

var (
	appInputs       = shared.NewInputs()
	usersFile       = "./configs/users.json"
	urlResolverPort = 4299
)

type (
	AccountUser struct {
		AccountName  string `json:"accountName"`
		UserName     string `json:"userName"`
		UserPassword string `json:"userPassword"`
	}
)

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
			services_user_strore_inmemory.AddSingletonUserStore(builder)
			ctn := builder.Build()

			userStore, err := di.TryGet[contracts_users.IUserStore](ctn)
			if err != nil {
				log.Error().Err(err).Msg("failed to get user store")
				return err
			}

			accountFetchUrlRoot := fmt.Sprintf("http://localhost:%d/jwt/v1/accounts/name/", urlResolverPort)

			getUsersResponse, err := userStore.GetUsers(ctx)
			printer.Println(cobra_utils.Blue, fluffycore_utils.PrettyJSON(getUsersResponse))

			// this creates a new account named as specified returning
			// the key used to sign users
			getOrCreateAccount := func(name string) (*contracts_nats.CreateSimpleAccountResponse, error) {

				var err error

				audFetchUrl := fmt.Sprintf("%s%s", accountFetchUrlRoot, name)
				respAud, err := http.Get(audFetchUrl)
				if err != nil {
					return nil, err
				}
				defer respAud.Body.Close()
				body, err := io.ReadAll(respAud.Body)
				if err != nil {
					fmt.Println("Error:", err)
					return nil, err
				}
				createSimpleAccountResponse := &contracts_nats.CreateSimpleAccountResponse{}
				err = json.Unmarshal(body, createSimpleAccountResponse)
				if err != nil {
					return nil, err
				}
				return createSimpleAccountResponse, nil
			}
			// load the callout key
			cKP, err := loadAndParseKeys(appInputs.CalloutIssuerNKeyFile, 'A')
			if err != nil {
				panic(fmt.Errorf("error loading callout issuer: %w", err))
			}

			// the authorizer function
			authorizer := func(req *jwt.AuthorizationRequest) (string, error) {
				// reading the account name from the token, likely this will be
				// encoded string with more information

				username := req.ConnectOptions.Username
				password := req.ConnectOptions.Password

				userParts := strings.Split(username, "@")
				username = userParts[0]
				var account string
				if len(userParts) > 1 {
					account = userParts[1]
				}
				accountUser := &AccountUser{
					AccountName:  strings.ToLower(account),
					UserName:     strings.ToLower(username),
					UserPassword: password,
				}
				log := log.With().Interface("accountUser", accountUser).Logger()

				authenticateUserResponse, err := userStore.AuthenticateUser(ctx,
					&contracts_users.AuthenticateUserRequest{
						UserName: accountUser.UserName,
						Password: accountUser.UserPassword,
						Account:  accountUser.AccountName,
					})

				if err != nil {
					log.Error().Err(err).Msg("user not allowed")
					return "", err
				}
				user := authenticateUserResponse.User
				// see if we have this account
				createSimpleAccountResponse, err := getOrCreateAccount(accountUser.AccountName)
				if err != nil {
					return "", err
				}

				// issue the user
				uc := jwt.NewUserClaims(req.UserNkey)
				// put the user in the global account
				uc.Audience = createSimpleAccountResponse.Audience
				// add whatever permissions you need
				uc.Sub.Allow.Add(user.Sub.Allow...)
				uc.Pub.Allow.Add(user.Pub.Allow...)

				uc.Sub.Deny.Add(user.Sub.Deny...)
				uc.Pub.Deny.Add(user.Pub.Deny...)

				// perhaps add an expiration to the JWT
				uc.Expires = time.Now().Unix() + 90
				kp, err := nkeys.FromSeed(createSimpleAccountResponse.KeyPair.Seed)
				if err != nil {
					return "", err
				}
				return uc.Encode(kp)
			}
			// connect the service with the creds
			opts, err := getConnectionOptions(appInputs.CalloutCreds)
			if err != nil {
				log.Error().Err(err).Msg("error loading creds")
				return err
			}
			nc, err := nats.Connect(appInputs.NatsUrl, opts...)
			if err != nil {
				log.Error().Err(err).Msg("error connecting")
				return err
			}
			defer nc.Close()
			// start the service
			_, err = callout.NewAuthorizationService(nc, callout.Authorizer(authorizer), callout.ResponseSignerKey(cKP))
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
	shared.InitCommonConnFlags(appInputs, command)

	flagName := "users.file"
	defaultS := usersFile
	command.Flags().StringVar(&usersFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "callout.issuer.nk"
	defaultS = "C.nk"
	command.Flags().StringVar(&appInputs.CalloutIssuerNKeyFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "callout.creds"
	defaultS = "service.creds"
	command.Flags().StringVar(&appInputs.CalloutCreds, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "url.resolver.port"
	defaultInt := urlResolverPort
	command.Flags().IntVar(&urlResolverPort, flagName, defaultInt, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	parentCmd.AddCommand(command)

}

func loadAndParseKeys(fp string, kind byte) (nkeys.KeyPair, error) {
	if fp == "" {
		return nil, errors.New("key file required")
	}
	seed, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("error reading key file: %w", err)
	}
	if !bytes.HasPrefix(seed, []byte{'S', kind}) {
		return nil, fmt.Errorf("key must be a private key")
	}
	kp, err := nkeys.FromSeed(seed)
	if err != nil {
		return nil, fmt.Errorf("error parsing key: %w", err)
	}
	return kp, nil
}

func getConnectionOptions(fp string) ([]nats.Option, error) {
	if fp == "" {
		return nil, errors.New("creds file required")
	}
	return []nats.Option{nats.UserCredentials(fp)}, nil
}
