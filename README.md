# AnonID issuer

An IRMA issuer of randomly generated usernames.

## Description

This project contains a HTTP server which issues an IRMA credential containing a random username, and the name of the organization on behalf of which the username was issued (the client), as follows:

1. The client starts the session by invoking `POST /session`, authenticating itself using a preshared token in the `Authorization` HTTP header.
2. Using a [builtin IRMA server library](https://irma.app/docs/irma-server-lib/), this server starts an issuance session for a credential containing a new random username, and the client's name as specified by the configuration (see below). It responds to the client with an [IRMA session package](https://irma.app/docs/api-irma-server/#post-session), who forwards it to its frontend. The [`irma-frontend`](https://irma.app/docs/irma-frontend/) library of the client's frontend uses that to show a QR code or universal link for the user's IRMA app.
4. The user's IRMA app and this server perform the issuance session.

## Getting Started

### Installing

```sh
git clone github.com/privacybydesign/anonid-issuer
cd anonid-issuer
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

    // Attribute IDs in which the username and client name are to be issued
    // Must be part of the same credential type
    "username_attr": "irma-demo.anonid.anonid.username",
    "client_attr": "irma-demo.anonid.anonid.website",
    
    "username_length": 12, // default value

    "listen_addr": "0.0.0.0",
    "port": 1234,
    "clients": {
        "SecretPresharedToken": "Client name (appears in 2nd attribute)"
    }
}
```

See also `conf.example.json`.

Note that this server does not itself support TLS. In production, it *must* be run behind a reverse proxy that handles TLS for it.

### Executing

```
anonid-issuer /path/to/config.json
```
