package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/bits"
	"net/http"

	"github.com/go-chi/chi"
	irma "github.com/privacybydesign/irmago"
	"github.com/privacybydesign/irmago/server"
	"github.com/privacybydesign/irmago/server/irmaserver"
)

const (
	// If d is the amount of possible usernames and n is the amount of generated usernames, then the
	// chance p that one or more usernames is generated twice ore more is approximately the
	// following, according to the generalized birthday problem
	// (https://en.wikipedia.org/wiki/Birthday_problem):
	//    p = 1 - e^(-n^2/(2d))
	// 12 characters using 62 characters gives d = 62^12. For n = 10^6, this results in a chance of
	// about 1.55 in 10 billion of duplicates:
	//    p = 1 - e^(-(10^6)^2/(2*62^12)) = 1.55 * 10^(-10)
	usernameDefaultLength = 12

	usernameCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type Server struct {
	conf *Configuration
}

func start(conf *Configuration) error {
	s := &Server{
		conf: conf,
	}

	handler := chi.NewMux()
	if s.conf.Verbose == 2 {
		handler.Use(server.LogMiddleware("anonid-issuer", server.LogOptions{Response: true}))
	}
	handler.Mount("/irma/", irmaserver.HandlerFunc())
	handler.Post("/session", s.handleSession)

	fullAddr := fmt.Sprintf("%s:%d", conf.ListenAddress, conf.Port)
	return server.FilterStopError(http.ListenAndServe(fullAddr, handler))
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	client, ok := s.conf.Clients[auth]
	if !ok {
		s.conf.Logger.WithField("authorization", auth).Warn("received request with unknown authorization")
		w.WriteHeader(401)
		return
	}

	s.conf.Logger.WithField("client", client).Info("handling request")

	username, err := newUsername(s.conf.UsernameLength)
	if err != nil {
		_ = server.LogError(err)
		w.WriteHeader(500)
		return
	}

	bts, err := s.startSession(username, client)
	if err != nil {
		_ = server.LogError(err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(bts)
	if err != nil {
		_ = server.LogError(err)
	}
}

func (s *Server) startSession(username, client string) ([]byte, error) {
	credid := s.conf.UsernameAttr.CredentialTypeIdentifier()
	request := irma.NewIssuanceRequest([]*irma.CredentialRequest{{
		CredentialTypeID: credid,
		Attributes: map[string]string{
			s.conf.UsernameAttr.Name(): username,
			s.conf.ClientAttr.Name():   client,
		},
	}})

	// Start the session. Discard the frontend token: we don't use it in this server, this server
	// does not expose the requestor API taking this token as parameter.
	sesPtr, _, frontendRequest, err := irmaserver.StartSession(request, nil)
	if err != nil {
		return nil, err
	}

	bts, err := json.Marshal(server.SessionPackage{
		SessionPtr:      sesPtr,
		FrontendRequest: frontendRequest,
	})
	if err != nil {
		return nil, err
	}

	return bts, nil
}

// randomNumbers returns a slice of integers below the specified maximum of the specified length,
// where each number between 0 and max-1 (inclusive) is chosen with equal probability.
func randomNumbers(length uint, max uint8) ([]uint8, error) {
	var (
		ints = make([]uint8, 0, length)
		bts  []byte
		num  uint8
		mask uint8 = 1<<bits.Len8(max) - 1 // in binary: "bits.Len8(max)" consecutive 1's
	)

	for len(ints) < int(length) {
		// generate a bunch of new random bytes if we've run out
		if len(bts) == 0 {
			bts = make([]byte, 10)
			_, err := rand.Read(bts)
			if err != nil {
				return nil, err
			}
		}

		num, bts = bts[0], bts[1:] // pop off the first byte
		num &= mask                // throw away unnecessary bits

		if num >= max {
			continue
		}
		ints = append(ints, num)
	}

	return ints, nil
}

func newUsername(length uint) (string, error) {
	r, err := randomNumbers(length, byte(len(usernameCharset)))
	if err != nil {
		return "", err
	}

	b := make([]byte, length)
	for i := range b {
		b[i] = usernameCharset[r[i]]
	}
	return string(b), nil
}
