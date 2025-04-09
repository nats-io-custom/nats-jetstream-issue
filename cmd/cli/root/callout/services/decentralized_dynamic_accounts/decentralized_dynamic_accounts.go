package decentralized_dynamic_accounts

import (
	"bytes"
	"errors"
	"fmt"
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
	services_account_store_golang_db "github.com/nats-io-custom/nats-jetstream-issue/internal/services/account_store/golang_db"
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

const use = "decentralized_dynamic_accounts"

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

			authAccountJWT, err := os.ReadFile(appInputs.AuthAccountJWTFile)
			if err != nil {
				return err
			}
			systemAccountJWT, err := os.ReadFile(appInputs.SystemAccountJWTFile)
			if err != nil {
				return err
			}
			okp, err := loadAndParseKeys(appInputs.OperatorNKeyFile, 'O')
			if err != nil {
				log.Error().Err(err).Msg("error loading operator key")
				return err
			}
			seed, _ := okp.Seed()

			sysOpts, err := getConnectionOptions(appInputs.SysCredsFile)
			if err != nil {
				log.Error().Err(err).Msg("error loading creds")
				return err
			}

			builder := di.Builder()

			di.AddInstance[*contracts_users.UserStoreConfig](builder,
				&contracts_users.UserStoreConfig{
					UserFile: usersFile,
				})
			di.AddInstance[*contracts_nats.AccountStoreConfig](builder,
				&contracts_nats.AccountStoreConfig{
					SystemAccountJWT: string(systemAccountJWT),
					AuthAccountJWT:   string(authAccountJWT),
					OperatorNKey:     seed,
				})

			services_user_strore_inmemory.AddSingletonUserStore(builder)
			services_account_store_golang_db.AddSingletonAccountStore(builder)
			ctn := builder.Build()

			accountStore, err := di.TryGet[contracts_nats.IAccountStore](ctn)
			if err != nil {
				log.Error().Err(err).Msg("failed to get account store")
				return err
			}
			accountStore.AddAccountByJWT(ctx,
				&contracts_nats.AddAccountByJWTRequest{
					Name: "auth",
					JWT:  string(authAccountJWT),
				})
			accountStore.AddAccountByJWT(ctx,
				&contracts_nats.AddAccountByJWTRequest{
					Name: "system",
					JWT:  string(systemAccountJWT),
				})

			userStore, err := di.TryGet[contracts_users.IUserStore](ctn)
			if err != nil {
				log.Error().Err(err).Msg("failed to get user store")
				return err
			}

			getUsersResponse, err := userStore.GetUsers(ctx)
			printer.Println(cobra_utils.Blue, fluffycore_utils.PrettyJSON(getUsersResponse))

			sysConn, err := nats.Connect(appInputs.NatsUrl, sysOpts...)
			if err != nil {
				log.Error().Err(err).Msg("error connecting to nats")
				return err
			}
			_, err = sysConn.Subscribe("$SYS.REQ.ACCOUNT.*.CLAIMS.LOOKUP",
				func(m *nats.Msg) {
					chunks := strings.Split(m.Subject, ".")
					id := chunks[3]
					fmt.Println(id)
					getAccountByPublicKeyResponse, err := accountStore.GetAccountByPublicKey(ctx,
						&contracts_nats.GetAccountByPublicKeyRequest{
							PublicKey: id,
						})

					if err == nil {
						accountInfo := getAccountByPublicKeyResponse.AccountInfo
						err = m.Respond([]byte(accountInfo.JWT))
						if err != nil {
							log.Error().Err(err).Msg("error responding")
						}
						refreshJWTResponse, err := accountStore.RefreshJWT(ctx,
							&contracts_nats.RefreshJWTRequest{
								IssuerKeyPair: okp,
								AccountInfo:   accountInfo,
							})
						if err != nil {
							log.Error().Err(err).Msg("error refreshing jwt")
							return
						}
						jwt := refreshJWTResponse.AccountInfo.JWT
						err = m.Respond([]byte(accountInfo.JWT))
						if err != nil {
							log.Error().Err(err).Msg("error responding")
						}
						printer.Print(cobra_utils.Blue, jwt)
					} else {
						_ = m.Respond(nil)
					}

				})
			if err != nil {
				log.Error().Err(err).Msg("error subscribing to claims lookup")
				return err
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

				getAccountByNameResponse, err := accountStore.GetAccountByName(ctx,
					&contracts_nats.GetAccountByNameRequest{
						Name: accountUser.AccountName,
					})
				if err != nil {
					log.Error().Err(err).Msgf("failed to get account by name: %s", accountUser.AccountName)
					return "", err
				}
				accountInfo := getAccountByNameResponse.AccountInfo

				// issue the user
				uc := jwt.NewUserClaims(req.UserNkey)
				// put the user in the global account
				uc.Audience = accountInfo.Audience
				// add whatever permissions you need
				uc.Sub.Allow.Add(user.Sub.Allow...)
				uc.Pub.Allow.Add(user.Pub.Allow...)

				uc.Sub.Deny.Add(user.Sub.Deny...)
				uc.Pub.Deny.Add(user.Pub.Deny...)

				// perhaps add an expiration to the JWT
				uc.Expires = time.Now().Unix() + 90
				kp, err := nkeys.FromSeed(accountInfo.KeyPair.Seed)
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

	flagName = "sys.creds"
	defaultS = "sys.creds"
	command.Flags().StringVar(&appInputs.SysCredsFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "auth.account.jwt"
	defaultS = "auth.account.jwt"
	command.Flags().StringVar(&appInputs.AuthAccountJWTFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "system.account.jwt"
	defaultS = "system.account.jwt"
	command.Flags().StringVar(&appInputs.SystemAccountJWTFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "operator.nk"
	defaultS = "operator.nk"
	command.Flags().StringVar(&appInputs.OperatorNKeyFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
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
