package nats

import (
	"context"

	jwt "github.com/nats-io/jwt/v2"
	nkeys "github.com/nats-io/nkeys"
)

type (
	AccountStoreConfig struct {
		OperatorNKey     []byte `json:"-"`
		SystemAccountJWT string `json:"-"`
		AuthAccountJWT   string `json:"-"`
	}

	GetAccountByNameRequest struct {
		Name string `json:"name"`
	}
	GetAccountByNameResponse struct {
		AccountInfo *CreateSimpleAccountResponse `json:"account_info"`
	}
	GetAccountByPublicKeyRequest struct {
		PublicKey string `json:"public_key"`
	}
	GetAccountByPublicKeyResponse struct {
		AccountInfo *CreateSimpleAccountResponse `json:"account_info"`
	}
	AddAccountByJWTRequest struct {
		Name string `json:"name"`
		JWT  string `json:"jwt"`
	}
	AddAccountByJWTResponse struct {
		AccountInfo *CreateSimpleAccountResponse `json:"account_info"`
	}

	GetAccountsResponse struct {
		Accounts []*CreateSimpleAccountResponse `json:"accounts"`
	}

	CreateSimpleAccountRequest struct {
		Name          string        `json:"name"`
		IssuerKeyPair nkeys.KeyPair `json:"issuer_key_pair"`
	}
	UpdateSimpleAccountRequest struct {
		Original      *CreateSimpleAccountResponse `json:"original"`
		IssuerKeyPair nkeys.KeyPair                `json:"issuer_key_pair"`
	}
	RawKeyPair struct {
		PublicKey  string `json:"public_key"`
		PrivateKey []byte `json:"private_key"`
		Seed       []byte `json:"seed"`
	}
	CommonAccountData struct {
		Name    string     `json:"name"`
		KeyPair RawKeyPair `json:"key_pair"`
		// JWT must assume will always be expired
		JWT string `json:"jwt"`
		// Audience is either a well known name that is in the static config like "SYS", or a public key id
		Audience      string             `json:"audience"`
		AccountClaims *jwt.AccountClaims `json:"account_claims"`
	}
	CreateSimpleAccountResponse struct {
		CommonAccountData
	}
	RefreshJWTRequest struct {
		IssuerKeyPair nkeys.KeyPair                `json:"issuer_key_pair"`
		AccountInfo   *CreateSimpleAccountResponse `json:"account_info"`
	}
	RefreshJWTResponse struct {
		AccountInfo *CreateSimpleAccountResponse `json:"account_info"`
	}
	IAccountStore interface {
		AddAccountByJWT(ctx context.Context, request *AddAccountByJWTRequest) (*AddAccountByJWTResponse, error)
		GetAccountByName(ctx context.Context, request *GetAccountByNameRequest) (*GetAccountByNameResponse, error)
		GetAccountByPublicKey(ctx context.Context, request *GetAccountByPublicKeyRequest) (*GetAccountByPublicKeyResponse, error)
		GetAccounts(ctx context.Context) (*GetAccountsResponse, error)
		RefreshJWT(ctx context.Context, request *RefreshJWTRequest) (*RefreshJWTResponse, error)
	}
)
