package mtproto

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/itsh4ro5/botupdate/internal/database"
)

type Service struct {
	apiID   int
	apiHash string
	store   database.Store

	authMu    sync.RWMutex
	authState *AuthState
}

type TermAuth struct {
	phone         string
	code          string
	password      string
	phoneChan     chan string
	codeChan      chan string
	passChan      chan string
	errChan       chan error
	onPasswordReq func()
}

func (t *TermAuth) Phone(_ context.Context) (string, error) {
	select {
	case p := <-t.phoneChan:
		return p, nil
	case err := <-t.errChan:
		return "", err
	}
}

func (t *TermAuth) Password(_ context.Context) (string, error) {
	if t.onPasswordReq != nil {
		t.onPasswordReq()
	}
	select {
	case p := <-t.passChan:
		return p, nil
	case err := <-t.errChan:
		return "", err
	}
}

func (t *TermAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (t *TermAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	select {
	case c := <-t.codeChan:
		return c, nil
	case err := <-t.errChan:
		return "", err
	}
}

func (t *TermAuth) SignUp(_ context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("signup not supported")
}

func NewService(apiID int, apiHash string, store database.Store) *Service {
	return &Service{
		apiID:   apiID,
		apiHash: apiHash,
		store:   store,
	}
}
