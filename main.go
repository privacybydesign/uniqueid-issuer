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

type Configuration struct {
	*server.Configuration

	ClientAttr   irma.AttributeTypeIdentifier `json:"client_attr"`
	UsernameAttr irma.AttributeTypeIdentifier `json:"username_attr"`

	UsernameLength uint `json:"username_length"`

	ListenAddress string            `json:"listen_addr"`
	Port          uint              `json:"port"`
	Clients       map[string]string `json:"clients"`
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
		if client == "" {
			die(fmt.Sprintf("client with authorization token %s has empty name", auth), nil)
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

	if err := irmaserver.Initialize(conf.Configuration); err != nil {
		die("failed to configure IRMA server", err)
	}

	if conf.ClientAttr.CredentialTypeIdentifier() != conf.UsernameAttr.CredentialTypeIdentifier() {
		msg := fmt.Sprintf("attributes %s and %s do not belong to the same credential type", conf.ClientAttr, conf.UsernameAttr)
		die(msg, nil)
	}

	irmaconf := conf.Configuration.IrmaConfiguration
	credid := conf.ClientAttr.CredentialTypeIdentifier()
	credtype := irmaconf.CredentialTypes[credid]
	if credtype == nil {
		die("nonexistent credential type: "+credid.String(), nil)
	}
	for _, attr := range []irma.AttributeTypeIdentifier{conf.ClientAttr, conf.UsernameAttr} {
		if !credtype.ContainsAttribute(attr) {
			die(fmt.Sprintf("credential type %s has no attribute %s", credid, attr.Name()), nil)
		}
	}

	if conf.UsernameLength == 0 {
		conf.UsernameLength = usernameDefaultLength
	}
}

func die(message string, err error) {
	if err != nil {
		message = fmt.Sprintf("%s: %s", message, err)
	}

	logger.Error(message)
	os.Exit(1)
}
