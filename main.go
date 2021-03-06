package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	irma "github.com/privacybydesign/irmago"
	"github.com/privacybydesign/irmago/server"
	"github.com/privacybydesign/irmago/server/irmaserver"
)

var logger = server.NewLogger(0, false, false)

const authTokenMinLength = 20

type Configuration struct {
	*server.Configuration

	ClientAttr    irma.AttributeTypeIdentifier `json:"client_attr"`
	LoginCodeAttr irma.AttributeTypeIdentifier `json:"logincode_attr"`

	LoginCodeLength uint `json:"logincode_length"`

	ListenAddress string            `json:"listen_addr"`
	Port          uint              `json:"port"`
	Clients       map[string]Client `json:"clients"`

	// TLS configuration
	TLSCertificateFile string `json:"tls_cert_file"`
	TLSPrivateKeyFile  string `json:"tls_privkey_file"`
}

type Client struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

func main() {
	if len(os.Args) != 2 {
		msg := fmt.Sprintf("%d arguments received, 1 expected: pass path to JSON configuration file as argument", len(os.Args)-1)
		die(msg, nil)
	}

	bts, err := os.ReadFile(os.Args[1])
	if err != nil {
		die("failed to read configuration file", err)
	}

	conf := new(Configuration)
	if err = json.Unmarshal(bts, conf); err != nil {
		die("failed to unmarshal configuration", err)
	}
	conf.Logger = logger
	logger.Level = server.Verbosity(conf.Verbose)

	checkConfig(conf)

	if err = start(conf); err != nil {
		die("failed to start server", err)
	}
}

func checkConfig(conf *Configuration) {
	if len(conf.Clients) == 0 {
		die("no clients configured", nil)
	}
	for auth, client := range conf.Clients {
		if client.Name == "" {
			die(fmt.Sprintf("client with authorization token %s has empty name", auth), nil)
		}
		if client.Domain == "" {
			die(fmt.Sprintf("client %s has empty domain name", client.Name), nil)
		}
		if len(auth) < authTokenMinLength {
			msg := fmt.Sprintf(
				"client %s has authentication token of length %d, should have at least length %d",
				client, len(auth), authTokenMinLength,
			)
			die(msg, nil)
		}
	}

	if conf.URL == "" {
		localIP, err := server.LocalIP()
		if err != nil {
			die("failed to determine local IP", err)
		}
		conf.URL = fmt.Sprintf("http://%s:%d/irma", localIP, conf.Port)
	} else {
		// conf.URL has to point to the endpoints for the IRMA app which are mounted at /irma/
		if !strings.HasSuffix(conf.URL, "/") {
			conf.URL += "/"
		}
		if !strings.HasSuffix(conf.URL, "irma/") {
			conf.URL += "irma/"
		}
	}

	if (conf.TLSPrivateKeyFile != "" && conf.TLSCertificateFile == "") ||
		(conf.TLSCertificateFile != "" && conf.TLSPrivateKeyFile == "") {
		die("either configure both or none of tls_cert_file and tls_privkey_file", nil)
	}

	if err := irmaserver.Initialize(conf.Configuration); err != nil {
		die("failed to configure IRMA server", err)
	}

	if conf.ClientAttr.CredentialTypeIdentifier() != conf.LoginCodeAttr.CredentialTypeIdentifier() {
		msg := fmt.Sprintf("attributes %s and %s do not belong to the same credential type", conf.ClientAttr, conf.LoginCodeAttr)
		die(msg, nil)
	}

	irmaconf := conf.Configuration.IrmaConfiguration
	credid := conf.ClientAttr.CredentialTypeIdentifier()
	credtype := irmaconf.CredentialTypes[credid]
	if credtype == nil {
		die("nonexistent credential type: "+credid.String(), nil)
	}
	for _, attr := range []irma.AttributeTypeIdentifier{conf.ClientAttr, conf.LoginCodeAttr} {
		if !credtype.ContainsAttribute(attr) {
			die(fmt.Sprintf("credential type %s has no attribute %s", credid, attr.Name()), nil)
		}
	}

	if conf.LoginCodeLength == 0 {
		conf.LoginCodeLength = loginCodeDefaultLength
	}
}

func (conf *Configuration) clientDomains() []string {
	domains := make([]string, 0, len(conf.Clients))
	for _, client := range conf.Clients {
		domains = append(domains, client.Domain)
	}
	return domains
}

func die(message string, err error) {
	if err != nil {
		message = fmt.Sprintf("%s: %s", message, err)
	}

	logger.Error(message)
	os.Exit(1)
}
