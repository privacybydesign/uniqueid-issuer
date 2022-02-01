# UniqueID issuer

An IRMA issuer of randomly generated login codes that can be used as unique random usernames in for example a website.

## Description

This project contains an HTTP server which issues an IRMA credential containing a random login code, and the name of the organization on behalf of which the login code was issued (the client), as follows:

1. The client starts the session by invoking `POST /session`, authenticating itself using a pre-shared token in the `Authorization` HTTP header.
2. Using a [builtin IRMA server library](https://irma.app/docs/irma-server-lib/), this server starts an issuance session for a credential containing a new random login code, and the client's organization name as specified by the configuration (see below). It responds to the client with an [IRMA session package](https://irma.app/docs/api-irma-server/#post-session), who forwards it to its frontend. The [`irma-frontend`](https://irma.app/docs/irma-frontend/) library of the client's frontend uses that to show a QR code or universal link for the user's IRMA app.
3. The user's IRMA app and this server perform the issuance session.

## Getting Started

### Installing

```sh
git clone github.com/privacybydesign/uniqueid-issuer
cd uniqueid-issuer
go install
```

### Configuring

Using a JSON configuration file:

```json5
{
    // Any JSON-serialized member of the IRMA library configuration struct is accepted
    // https://pkg.go.dev/github.com/privacybydesign/irmago/server#Configuration
    // For example:
    "schemes_path": "/path/to/schemes", // optional
    "url": "https://example.com/",      // public URL to this server

    // Attribute IDs in which the login code and client name are to be issued
    // Must be part of the same credential type
    "logincode_attr": "irma-demo.sidn-pbdf.uniqueid.uniqueid",
    "client_attr": "irma-demo.sidn-pbdf.uniqueid.organization",
    
    "logincode_length": 12, // default value

    "listen_addr": "0.0.0.0",
    "port": 1234,
    "clients": {
        "SecretPresharedToken": {
            "name": "Client name (appears in 2nd attribute)",
            "domain": "https://example-client.com/"
        }
    },
    
    // optional (specify either both or none)
    // if present, enables TLS
    "tls_cert_file": "/path/to/cert.pem",
    "tls_privkey_file": "/path/to/privkey.pem"
}
```

See also `conf.example.json`.

The TLS configuration fields are optional. If present, the server will accept TLS connection. If not, in production this server *must* be run behind a reverse proxy that handles TLS for it.

### Executing

You can run the issuer as follows.

    uniqueid-issuer /path/to/config.json

Subsequently, you can start an issuing session using the `irma` CLI tool.

    go install github.com/privacybydesign/irmago/irma@master
    irma session --from-package $(curl -X POST -H "Authorization: SecretPresharedToken" http://localhost:1234/session)
