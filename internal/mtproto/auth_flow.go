package mtproto

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
)

type AuthState struct {
	mu         sync.Mutex
	cancel     context.CancelFunc
	term       *TermAuth
	inProgress bool
	isPassword bool
}

// StartAuth begins the MTProto authentication loop in a background goroutine.
func (s *Service) StartAuth(ctx context.Context, phone string, onPasswordReq func()) error {
	s.authMu.Lock()
	if s.authState != nil && s.authState.inProgress {
		s.authMu.Unlock()
		return errors.New("authentication already in progress")
	}

	authCtx, cancel := context.WithCancel(context.Background())
	term := &TermAuth{
		phoneChan:     make(chan string, 1),
		codeChan:      make(chan string, 1),
		passChan:      make(chan string, 1),
		errChan:       make(chan error, 1),
		onPasswordReq: onPasswordReq,
	}
	term.phoneChan <- phone

	s.authState = &AuthState{
		cancel:     cancel,
		term:       term,
		inProgress: true,
	}
	s.authMu.Unlock()

	// Memory session storage
	sessionStorage := &session.StorageMemory{}

	// Create client
	client := telegram.NewClient(s.apiID, s.apiHash, telegram.Options{
		SessionStorage: sessionStorage,
	})

	go func() {
		defer func() {
			s.authMu.Lock()
			if s.authState != nil {
				s.authState.inProgress = false
			}
			s.authMu.Unlock()
		}()

		// Run the client and start auth flow
		err := client.Run(authCtx, func(ctx context.Context) error {
			flow := auth.NewFlow(term, auth.SendCodeOptions{})
			if err := client.Auth().IfNecessary(ctx, flow); err != nil {
				// We can catch password required here if the flow fails and pass it back
				// but gotd handles password internally via term.Password()
				return err
			}

			// Successfully authenticated!
			// Export session
			loader := session.Loader{Storage: sessionStorage}
			data, err := loader.Load(ctx)
			if err != nil {
				return err
			}

			// Save to main DB
			state, err := s.store.Load(ctx)
			if err == nil {
				jsonBytes, _ := json.Marshal(data)
				state.UserbotSession = string(jsonBytes)
				state.UserbotPhone = phone
				s.store.Save(ctx, state)
			}
			return nil
		})

		if err != nil {
			term.errChan <- err
		}
	}()

	return nil
}

func (s *Service) SubmitOTP(otp string) error {
	s.authMu.RLock()
	state := s.authState
	s.authMu.RUnlock()

	if state == nil || !state.inProgress {
		return errors.New("no authentication in progress")
	}
	state.term.codeChan <- otp
	return nil
}

func (s *Service) SubmitPassword(password string) error {
	s.authMu.RLock()
	state := s.authState
	s.authMu.RUnlock()

	if state == nil || !state.inProgress {
		return errors.New("no authentication in progress")
	}
	state.term.passChan <- password
	return nil
}

func (s *Service) CancelAuth() {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	if s.authState != nil {
		s.authState.cancel()
		s.authState.inProgress = false
	}
}
