package inmemory

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	di "github.com/fluffy-bunny/fluffy-dozm-di"
	fluffycore_utils "github.com/fluffy-bunny/fluffycore/utils"
	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"
	jwt "github.com/nats-io/jwt/v2"
	nkeys "github.com/nats-io/nkeys"
	zerolog "github.com/rs/zerolog"
)

type (
	service struct {
		lock                             sync.Mutex
		config                           *contracts_nats.AccountStoreConfig
		accountFriendlyNameToAccountInfo map[string]*contracts_nats.CreateSimpleAccountResponse
		accountPubKeyToAccountInfo       map[string]*contracts_nats.CreateSimpleAccountResponse
		issuerKeyPair                    nkeys.KeyPair
	}
)

var stemService = (*service)(nil)
var _ contracts_nats.IAccountStore = (*service)(nil)

func (s *service) Ctor(config *contracts_nats.AccountStoreConfig) (contracts_nats.IAccountStore, error) {

	kp, err := loadAndParseKeys([]byte(config.OperatorNKey), 'O')
	if err != nil {
		return nil, err
	}

	svc := &service{
		config:                           config,
		issuerKeyPair:                    kp,
		accountFriendlyNameToAccountInfo: map[string]*contracts_nats.CreateSimpleAccountResponse{},
		accountPubKeyToAccountInfo:       map[string]*contracts_nats.CreateSimpleAccountResponse{},
	}
	if fluffycore_utils.IsNotEmptyOrNil(config.SystemAccountJWT) {
		svc.AddAccountByJWT(context.Background(),
			&contracts_nats.AddAccountByJWTRequest{
				Name: "system",
				JWT:  config.SystemAccountJWT,
			})
	}
	if fluffycore_utils.IsNotEmptyOrNil(config.AuthAccountJWT) {
		svc.AddAccountByJWT(context.Background(),
			&contracts_nats.AddAccountByJWTRequest{
				Name: "auth",
				JWT:  config.SystemAccountJWT,
			})
	}
	return svc, nil
}
func AddSingletonAccountStore(builder di.ContainerBuilder) {
	di.AddSingleton[contracts_nats.IAccountStore](
		builder,
		stemService.Ctor,
	)
}
func (s *service) AddAccountByJWT(ctx context.Context, request *contracts_nats.AddAccountByJWTRequest) (*contracts_nats.AddAccountByJWTResponse, error) {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	log := zerolog.Ctx(ctx).With().Logger()

	subI, err := ExtractClaimFromJWTNoValidation(request.JWT, "sub")
	if err != nil {
		log.Error().Err(err).Msg("error extracting sub from jwt")
		return nil, err
	}
	subject, ok := subI.(string)
	if !ok {
		log.Error().Err(err).Msg("error converting sub to string")
		return nil, fmt.Errorf("error converting sub to string")
	}
	accountInfo, ok := s.accountPubKeyToAccountInfo[subject]
	if ok {
		// already here
		return &contracts_nats.AddAccountByJWTResponse{
			AccountInfo: accountInfo,
		}, nil
	}
	accountInfo = &contracts_nats.CreateSimpleAccountResponse{
		CommonAccountData: contracts_nats.CommonAccountData{
			Name:     request.Name,
			Audience: subject,
			JWT:      request.JWT,
		},
	}
	s.accountFriendlyNameToAccountInfo[request.Name] = accountInfo
	s.accountPubKeyToAccountInfo[subject] = accountInfo
	return &contracts_nats.AddAccountByJWTResponse{
		AccountInfo: accountInfo,
	}, nil
}

func (s *service) GetAccountByName(ctx context.Context, request *contracts_nats.GetAccountByNameRequest) (*contracts_nats.GetAccountByNameResponse, error) {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	log := zerolog.Ctx(ctx).With().Interface("request", request).Logger()
	var err error
	accountInfo, ok := s.accountFriendlyNameToAccountInfo[request.Name]
	if !ok {

		accountInfo, err = CreateSimpleAccount(ctx,
			&contracts_nats.CreateSimpleAccountRequest{
				Name:          request.Name,
				IssuerKeyPair: s.issuerKeyPair,
			})
		if err != nil {
			log.Error().Err(err).Msg("error creating account")
			return nil, err
		}

		s.accountFriendlyNameToAccountInfo[request.Name] = accountInfo
		s.accountPubKeyToAccountInfo[accountInfo.KeyPair.PublicKey] = accountInfo
	}
	log.Info().Interface("account_info", accountInfo).Msg("account found")

	return &contracts_nats.GetAccountByNameResponse{
		AccountInfo: accountInfo,
	}, nil
}

func (s *service) GetAccountByPublicKey(ctx context.Context, request *contracts_nats.GetAccountByPublicKeyRequest) (*contracts_nats.GetAccountByPublicKeyResponse, error) {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	log := zerolog.Ctx(ctx).With().Interface("request", request).Logger()

	accountInfo, ok := s.accountPubKeyToAccountInfo[request.PublicKey]
	if !ok {
		log.Error().Err(fmt.Errorf("account not found")).Msg("error getting account by public key")
		return nil, fmt.Errorf("account not found")
	}
	switch accountInfo.Name {
	case "system", "auth":
		return &contracts_nats.GetAccountByPublicKeyResponse{
			AccountInfo: accountInfo,
		}, nil
	}
	refreshJWTResponse, err := s.RefreshJWT(ctx, &contracts_nats.RefreshJWTRequest{
		IssuerKeyPair: s.issuerKeyPair,
		AccountInfo:   accountInfo,
	})
	if err != nil {
		log.Error().Err(err).Msg("error refreshing jwt")
		return nil, err
	}
	accountInfo = refreshJWTResponse.AccountInfo

	log.Info().Interface("account_info", accountInfo).Msg("account found")
	return &contracts_nats.GetAccountByPublicKeyResponse{
		AccountInfo: accountInfo,
	}, nil

}
func (s *service) GetAccounts(ctx context.Context) (*contracts_nats.GetAccountsResponse, error) {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//

	log := zerolog.Ctx(ctx).With().Logger()
	accounts := []*contracts_nats.CreateSimpleAccountResponse{}
	for k, v := range s.accountFriendlyNameToAccountInfo {
		log.Info().Msgf("account: %s", k)
		accounts = append(accounts, v)
	}
	return &contracts_nats.GetAccountsResponse{
		Accounts: accounts,
	}, nil
}
func (s *service) RefreshJWT(ctx context.Context, request *contracts_nats.RefreshJWTRequest) (*contracts_nats.RefreshJWTResponse, error) {

	log := zerolog.Ctx(ctx).With().Interface("request", request).Logger()

	request.AccountInfo.AccountClaims.Expires = time.Now().Add(time.Minute * 2).Unix()

	// now we could encode an issue the account using the operator
	// key that we generated above, but this will illustrate that
	// the account could be self-signed, and given to the operator
	// who can then re-sign it
	accountJWT, err := request.AccountInfo.AccountClaims.Encode(request.IssuerKeyPair)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode account")
		return nil, err
	}
	request.AccountInfo.JWT = accountJWT
	return &contracts_nats.RefreshJWTResponse{
		AccountInfo: request.AccountInfo,
	}, nil
}

// ParseJWT parses a JWT and extracts the 'sub' claim
func ExtractClaimFromJWTNoValidation(token string, claimName string) (interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decoding payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("unmarshalling payload: %v", err)
	}

	claimValue, ok := claims[claimName]
	if !ok {
		return "", fmt.Errorf("'sub' claim not found or invalid")
	}

	return claimValue, nil
}

func CreateSimpleAccount(ctx context.Context, request *contracts_nats.CreateSimpleAccountRequest) (*contracts_nats.CreateSimpleAccountResponse, error) {
	log := zerolog.Ctx(ctx).With().Str("func", "CreateSimpleAccount").Logger()
	// create an account keypair
	akp, err := nkeys.CreateAccount()
	if err != nil {
		log.Error().Err(err).Msg("failed to create account")
		return nil, err
	}
	// extract the public key for the account
	apk, err := akp.PublicKey()
	if err != nil {
		log.Error().Err(err).Msg("failed to get public key")
		return nil, err
	}
	/*
		askp, err := nkeys.CreateAccount()
		if err != nil {
			log.Error().Err(err).Msg("failed to create account")
			return nil, err
		}
		// extract the public key for the account
		aspk, err := askp.PublicKey()
		if err != nil {
			log.Error().Err(err).Msg("failed to get public key")
			return nil, err
		}
	*/

	// create the claim for the account using the public key of the account
	ac := jwt.NewAccountClaims(apk)
	ac.Name = request.Name
	ac.Expires = time.Now().Add(time.Minute * 2).Unix()
	ac.Limits.JetStreamLimits.DiskStorage = -1
	ac.Limits.JetStreamLimits.MemoryStorage = -1

	// add the signing key (public) to the account
	ac.SigningKeys.Add(apk)
	//ac.SigningKeys.Add(aspk)

	// now we could encode an issue the account using the operator
	// key that we generated above, but this will illustrate that
	// the account could be self-signed, and given to the operator
	// who can then re-sign it
	accountJWT, err := ac.Encode(request.IssuerKeyPair)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode account")
		return nil, err
	}

	resp := &contracts_nats.CreateSimpleAccountResponse{
		CommonAccountData: contracts_nats.CommonAccountData{
			Name:          request.Name,
			JWT:           accountJWT,
			AccountClaims: ac,
			Audience:      apk,
		},
	}
	resp.KeyPair.PublicKey, _ = akp.PublicKey()
	resp.KeyPair.PrivateKey, _ = akp.PrivateKey()
	resp.KeyPair.Seed, _ = akp.Seed()

	resp.Audience = resp.KeyPair.PublicKey
	return resp, nil
}

func loadAndParseKeys(seed []byte, kind byte) (nkeys.KeyPair, error) {

	if !bytes.HasPrefix(seed, []byte{'S', kind}) {
		return nil, fmt.Errorf("key must be a private key")
	}
	kp, err := nkeys.FromSeed(seed)
	if err != nil {
		return nil, fmt.Errorf("error parsing key: %w", err)
	}
	return kp, nil
}
