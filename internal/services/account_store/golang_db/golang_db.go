package golang_db

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	database "github.com/Mahopanda/golang-database/database"
	di "github.com/fluffy-bunny/fluffy-dozm-di"
	status "github.com/gogo/status"
	contracts_nats "github.com/nats-io-custom/nats-jetstream-issue/internal/contracts/nats"
	jwt "github.com/nats-io/jwt/v2"
	nkeys "github.com/nats-io/nkeys"
	zerolog "github.com/rs/zerolog"
	codes "google.golang.org/grpc/codes"
)

type (
	service struct {
		lock          sync.Mutex
		config        *contracts_nats.AccountStoreConfig
		issuerKeyPair nkeys.KeyPair
		lockManager   *lockManager
		db            *database.Driver
	}
	lockManager struct {
		mutexes map[string]*sync.Mutex
		lock    sync.Mutex
	}
)

func NewLockManager() *lockManager {
	return &lockManager{
		mutexes: make(map[string]*sync.Mutex),
	}
}
func (s *lockManager) GetLock(collection string) *sync.Mutex {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	if _, ok := s.mutexes[collection]; !ok {
		s.mutexes[collection] = &sync.Mutex{}
	}
	return s.mutexes[collection]
}

var stemService = (*service)(nil)
var _ contracts_nats.IAccountStore = (*service)(nil)

func (s *service) Ctor(config *contracts_nats.AccountStoreConfig) (contracts_nats.IAccountStore, error) {

	// Method 1: Using os.Executable()
	executablePath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	// build path to our db folder
	executableDir := filepath.Dir(executablePath)
	dbPath := filepath.Join(executableDir, "golang_db")
	logger := database.NewConsoleLogger()
	serializer := &database.JSONSerializer{}
	store := database.NewFileStore(dbPath, serializer)
	lockManager := NewLockManager()
	db := database.NewDriver(store, lockManager, logger)

	kp, err := loadAndParseKeys([]byte(config.OperatorNKey), 'O')
	if err != nil {
		return nil, err
	}

	return &service{
		config:        config,
		issuerKeyPair: kp,
		lockManager:   lockManager,
		db:            db,
	}, nil
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
	accountInfo, err := OneOf[contracts_nats.CreateSimpleAccountResponse](s.db, func(data map[string]interface{}) bool {
		return data["audience"] == subject
	})
	if err == nil && accountInfo != nil {
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
	err = s.db.Write("accounts", request.Name, accountInfo)
	if err != nil {
		log.Error().Err(err).Msg("error writing account")
		return nil, err
	}

	return &contracts_nats.AddAccountByJWTResponse{
		AccountInfo: accountInfo,
	}, nil
}
func ManyOf[T any](db *database.Driver, filter func(map[string]interface{}) bool) ([]*T, error) {
	var result []*T
	results, err := db.Query("accounts", filter)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, err
	}

	for _, v := range results {
		var item T
		jsondata, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsondata, &item)
		if err != nil {
			return nil, err
		}
		result = append(result, &item)
	}
	return result, nil
}
func OneOf[T any](db *database.Driver, filter func(map[string]interface{}) bool) (*T, error) {
	var result T
	results, err := db.Query("accounts", filter)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, status.Error(codes.NotFound, "not found")
	}
	jsondata, err := json.Marshal(results[0])
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsondata, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *service) GetAccountByName(ctx context.Context, request *contracts_nats.GetAccountByNameRequest) (*contracts_nats.GetAccountByNameResponse, error) {
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	s.lock.Lock()
	defer s.lock.Unlock()
	//--~--~--~--~-- BARBED WIRE --~--~--~--~--//
	log := zerolog.Ctx(ctx).With().Interface("request", request).Logger()
	var accountInfo *contracts_nats.CreateSimpleAccountResponse

	accountInfo, err := OneOf[contracts_nats.CreateSimpleAccountResponse](s.db, func(data map[string]interface{}) bool {
		return data["name"] == request.Name
	})

	if err != nil {
		// create a new account
		accountInfo, err = CreateSimpleAccount(ctx,
			&contracts_nats.CreateSimpleAccountRequest{
				Name:          request.Name,
				IssuerKeyPair: s.issuerKeyPair,
			})
		if err != nil {
			log.Error().Err(err).Msg("error creating account")
			return nil, err
		}
		err = s.db.Write("accounts", request.Name, accountInfo)
		if err != nil {
			log.Error().Err(err).Msg("error writing account")
			return nil, err
		}
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

	accountInfo, err := OneOf[contracts_nats.CreateSimpleAccountResponse](s.db, func(data map[string]interface{}) bool {
		return data["audience"] == request.PublicKey
	})

	if err != nil {
		log.Error().Err(err).Msg("error querying accounts")
		return nil, err
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
	accountInfos, err := ManyOf[contracts_nats.CreateSimpleAccountResponse](s.db, func(data map[string]interface{}) bool {
		return true
	})
	if err != nil {
		log.Error().Err(err).Msg("error querying accounts")
		return nil, err
	}

	return &contracts_nats.GetAccountsResponse{
		Accounts: accountInfos,
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
