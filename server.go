package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/bits"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
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
		handler.Use(server.LogMiddleware("unique-issuer", server.LogOptions{Response: true}))
	}

	handler.Use(cors.New(cors.Options{
		AllowedOrigins:   s.conf.clientDomains(),
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Cache-Control"},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
	}).Handler)

	handler.Mount("/irma/", irmaserver.HandlerFunc())
	handler.Post("/session", s.handleSession)

	fullAddr := fmt.Sprintf("%s:%d", conf.ListenAddress, conf.Port)

	if conf.TLSPrivateKey != "" {
		return server.FilterStopError(http.ListenAndServeTLS(fullAddr, conf.TLSCertificate, conf.TLSPrivateKey, handler))
	}
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

	s.conf.Logger.WithField("client", client.Name).Info("handling request")

	username, err := newUsername(s.conf.UsernameLength)
	if err != nil {
		_ = server.LogError(err)
		w.WriteHeader(500)
		return
	}

	bts, err := s.startSession(username, client.Name)
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
	// Set the expiry to 10 years from now; these attributes don't really expire in the usual sense
	validity := time.Now().AddDate(10, 0, 0)
	credid := s.conf.UsernameAttr.CredentialTypeIdentifier()
	request := irma.NewIssuanceRequest([]*irma.CredentialRequest{{
		CredentialTypeID: credid,
		Validity:         (*irma.Timestamp)(&validity),
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

// randomNumbers returns a slice of random integers of the length specified by the first parameter,
// each of them being smaller than the maximum specified by the second parameter, where each number
// between 0 and max-1 (inclusive) is chosen with equal probability.
func randomNumbers(length uint, max uint8) ([]uint8, error) {
	var (
		ints = make([]uint8, 0, length)
		bts  []byte
		num  uint8
		mask uint8 = 1<<bits.Len8(max) - 1 // in binary: "bits.Len8(max)" consecutive 1's
	)

	for len(ints) < int(length) {
		// Generate a bunch of new random bytes if we've run out.
		// We take 10 bytes every 10 iterations instead of 1 per iteration for efficiency.
		if len(bts) == 0 {
			bts = make([]byte, 10)
			_, err := rand.Read(bts)
			if err != nil {
				return nil, err
			}
		}

		num, bts = bts[0], bts[1:] // pop off the first byte, store it into num
		num &= mask                // throw away unnecessary bits

		if num >= max {
			continue
		}
		ints = append(ints, num)
	}

	return ints, nil
}

// newUsername generates a new username, that is a random string, where each character from the
// character set is chosen with equal probability.
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
