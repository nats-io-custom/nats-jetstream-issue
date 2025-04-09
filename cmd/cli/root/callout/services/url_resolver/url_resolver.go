package url_resolver

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	echo "github.com/labstack/echo/v4"
	cobra_utils "github.com/nats-io-custom/nats-jetstream-issue/internal/cobra_utils"
	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"

	//services_account_store_inmemory "github.com/nats-io-custom/nats-jetstream-issue/internal/services/account_store/inmemory"
	services_account_store_golang_db "github.com/nats-io-custom/nats-jetstream-issue/internal/services/account_store/golang_db"

	shared "github.com/nats-io-custom/nats-jetstream-issue/internal/shared"
	nkeys "github.com/nats-io/nkeys"
	zerolog "github.com/rs/zerolog"
	cobra "github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

const use = "url_resolver"

var (
	appInputs    = shared.NewInputs()
	accountMutex = sync.Mutex{}
	port         = 4299
)

var wellknownAccounts = []string{"svc", "edge"}

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
			builder := di.Builder()
			di.AddInstance[*contracts_nats.AccountStoreConfig](builder,
				&contracts_nats.AccountStoreConfig{
					SystemAccountJWT: string(systemAccountJWT),
					AuthAccountJWT:   string(authAccountJWT),
					OperatorNKey:     seed,
				})

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

			for _, wa := range wellknownAccounts {
				_, err = accountStore.GetAccountByName(ctx,
					&contracts_nats.GetAccountByNameRequest{
						Name: wa,
					})
				if err != nil {
					log.Error().Err(err).Msgf("failed to get account by name: %s", wa)
					return err
				}

			}

			e := echo.New()

			e.GET("/jwt/v1/accounts/id/", func(c echo.Context) error {
				r := c.Request()
				w := c.Response()
				ctx := r.Context()

				// full path with query string
				path := r.URL.Path + "?" + r.URL.RawQuery
				log := zerolog.Ctx(ctx).With().Str("command", path).Logger()
				log.Info().Send()

				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				w.WriteHeader(http.StatusOK)
				return nil
			})
			// Route to get account by ID
			e.GET("/jwt/v1/accounts/id/:id", func(c echo.Context) error {
				//--~--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
				accountMutex.Lock()
				defer accountMutex.Unlock()
				//--~--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
				id := c.Param("id")
				r := c.Request()
				// full path with query string
				path := r.URL.Path + "?" + r.URL.RawQuery
				log := zerolog.Ctx(ctx).With().Str("command", path).Logger()
				log.Info().Send()

				//	accountInfo, err := accountManager.GetAccountById(ctx, id)
				getAccountByPublicKeyResponse, err := accountStore.GetAccountByPublicKey(ctx,
					&contracts_nats.GetAccountByPublicKeyRequest{
						PublicKey: id,
					})
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error creating account: "+err.Error())
				}
				accountInfo := getAccountByPublicKeyResponse.AccountInfo
				theJWT := accountInfo.JWT
				if theJWT == "" {
					return c.String(http.StatusNotFound, "Account JWT not found")
				}

				return c.String(http.StatusOK, theJWT)
			})

			// Route to get account by name
			e.GET("/jwt/v1/accounts/name/:name", func(c echo.Context) error {

				//--~--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--
				accountMutex.Lock()
				defer accountMutex.Unlock()
				//--~--~--~--~--~--~-- BARBED WIRE --~--~--~--~--~--~--

				r := c.Request()
				// full path with query string
				path := r.URL.Path + "?" + r.URL.RawQuery

				log := zerolog.Ctx(ctx).With().Str("command", path).Logger()
				log.Info().Send()

				name := c.Param("name")
				name = strings.ToLower(name)

				getAccountByNameResponse, err := accountStore.GetAccountByName(ctx,
					&contracts_nats.GetAccountByNameRequest{
						Name: name,
					})

				//	info, err := accountManager.GetOrCreateAccountByFriendlyName(ctx, name)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error creating account: "+err.Error())
				}
				info := getAccountByNameResponse.AccountInfo

				// return just the id
				return c.JSON(http.StatusOK, info)
			})

			// Route to get all accounts
			e.GET("/jwt/v1/accounts/list", func(c echo.Context) error {

				accountResonse, err := accountStore.GetAccounts(ctx)
				if err != nil {
					return c.String(http.StatusInternalServerError, "Error creating account: "+err.Error())
				}
				return c.JSON(http.StatusOK, accountResonse)
			})

			address := fmt.Sprintf(":%d", port)
			printer.Infof("Starting server on %s", address)
			// Start the server on port 8080
			e.Start(address)
			return nil
		},
	}
	shared.InitCommonConnFlags(appInputs, command)

	flagName := "operator.nk"
	defaultS := "operator.nk"
	command.Flags().StringVar(&appInputs.OperatorNKeyFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "port"
	defaultInt := port
	command.Flags().IntVar(&defaultInt, flagName, defaultInt, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "auth.account.jwt"
	defaultS = "auth.account.jwt"
	command.Flags().StringVar(&appInputs.AuthAccountJWTFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
	viper.BindPFlag(flagName, command.PersistentFlags().Lookup(flagName))

	flagName = "system.account.jwt"
	defaultS = "system.account.jwt"
	command.Flags().StringVar(&appInputs.SystemAccountJWTFile, flagName, defaultS, fmt.Sprintf("[required] i.e. --%s=%s", flagName, defaultS))
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
