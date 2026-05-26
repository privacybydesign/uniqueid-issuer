# UniqueID issuer

A [Yivi](https://yivi.app) (formerly IRMA) issuer for one specific credential: a random identifier, paired with the name of the organisation that issued it. Users keep that credential in their Yivi app and disclose it later whenever that organisation needs to recognise them as a returning user, without ever learning who they are.

In production the credential is `pbdf.sidn-pbdf.uniqueid` and this server is run by SIDN under the Privacy by Design Foundation. The source is open so the implementation can be audited, and so anyone can run their own instance against `irma-demo` for testing, or against `pbdf` with their own credential type if the Foundation has approved one.

## Use cases

You want this issuer if you run a service and need a stable, pseudonymous identifier for the people interacting with it: a user scans Yivi, comes back later, and you recognise them as the same user, without ever learning who they are. The identifier is persistent so you can store it and key your records off it. It carries no personal data, and each user gets a different identifier at every organisation that issues through this server, so a user's activity can't be correlated across organisations.

The canonical application is login, replacing a username and password with a Yivi disclosure. The underlying primitive is broader than that, though: it works anywhere a unique id that you issue yourself would be useful. Examples:

- Login or account creation without collecting personal data
- Anonymous feedback or surveys where the same person can submit, edit, and follow up
- Whistleblower or tip lines that need continuity of identity without identifying the tipster
- Pseudonymous comments, voting, or recurring support tickets the user can come back to

The flow is always the same:

1. Your backend calls `POST /session` on this issuer with its preshared token in the `Authorization` header.
2. The issuer starts an IRMA issuance session and returns a [session package](https://irma.app/docs/api-irma-server/#post-session). Your backend forwards it to your frontend.
3. [`irma-frontend`](https://irma.app/docs/irma-frontend/) renders it as a QR code or universal link. The user scans it, and the Yivi app stores the credential. The credential contains a fresh random identifier and your organisation's name.
4. Whenever the user comes back, you ask them to disclose the credential. Verifying the disclosure is the job of a separate IRMA verifier; that is not part of this project.

This project is not a verifier. If you only need to check attributes, run an IRMA server in verifier mode instead. It is also not a generic issuer. The only thing it ever issues is an `(identifier, organisation)` pair.

You also may not need it if you already plan to identify users via an existing attribute like email. Verifying an attribute is simpler, but the same value comes back at every service that asks for it, so users become linkable across services by anyone with two databases, and a leak hands the attacker real email addresses. The uniqueid credential gives each organisation a random identifier instead, so neither problem applies.

## Schemes

Every Yivi credential type lives in a *scheme*: a signed bundle of issuer public keys and credential-type definitions that the Yivi app and every IRMA server download automatically. Two schemes matter here. `irma-demo` is the public test scheme; all of its private keys are published, so anyone can run an issuer for `irma-demo.sidn-pbdf.uniqueid` while developing. `pbdf` is the production scheme managed by the Privacy by Design Foundation; issuers and credential types are added only after vetting.

If you are using this issuer (rather than running your own), you do not need to do anything with the scheme. The credential type `pbdf.sidn-pbdf.uniqueid` is already registered, the Yivi app already knows it, and your job is simply to call `POST /session` and embed `irma-frontend`.

If you want to run your own deployment of this issuer, what you have to do depends on which scheme you target:

- Against `irma-demo`: nothing. The private key ships with the scheme. The example config already points at it.
- Against `pbdf`, re-issuing `pbdf.sidn-pbdf.uniqueid`: you need the production private key for the `sidn-pbdf` issuer, which is held by SIDN. In practice that means you are SIDN.
- Against `pbdf` under your own organisation: register your own issuer (and probably your own credential type) with `pbdf` through the Privacy by Design Foundation. The current procedure is roughly to open a PR against `irma-demo` that defines your issuer and credential type, get it merged for testing, then sign a contract with the Foundation and submit your `pbdf` keypair through their onboarding. See [docs.yivi.app](https://docs.yivi.app/) for the up-to-date process. Once you are in, point `logincode_attr` and `client_attr` at the attributes inside your own credential type.

The two attributes named in the config (`logincode_attr` and `client_attr`) must both live on the same credential type, and that credential type must exist in a scheme the embedded IRMA server has loaded. Otherwise the server refuses to start.

## Preshared tokens

Each entry under `clients` in the config is a service that's allowed to use this issuer. The map key is that client's preshared token: the secret it sends in the `Authorization` header on `POST /session`. The value holds the organisation name (which goes into the issued credential as the `organization` attribute) and the domain (used to build the CORS allowlist).

```json
"clients": {
    "<preshared token>": {
        "name": "Example Org",
        "domain": "https://example.com"
    }
}
```

Tokens must be at least 20 characters. There is no upper bound. Generate them with a CSPRNG, for example `openssl rand -base64 32` or `head -c 32 /dev/urandom | base64`. Use one token per client. To rotate, add the new entry, deploy, then remove the old one.

A token is a shared secret between you (the operator) and one specific client. You generate it, hand it over out of band (password manager, secrets vault), and the client stores it on its own backend. It must never reach the browser. The `name` is what users will see in the credential as the issuing organisation, so use something recognisable. The `domain` is what the issuer puts in `Access-Control-Allow-Origin`, so it has to match the origin the browser uses.

On the client's side, implementing this is one request:

```
POST /session
Authorization: <preshared token>
```

The JSON response goes to `irma-frontend`. The token stays on the server.

## Getting started

### Installing

```sh
git clone github.com/privacybydesign/uniqueid-issuer
cd uniqueid-issuer
go install
```

### Configuring

The config is a JSON file. Any field from the [IRMA server configuration struct](https://pkg.go.dev/github.com/privacybydesign/irmago/server#Configuration) is accepted, alongside the fields specific to this issuer:

```json5
{
    // Any JSON-serialized member of the IRMA library configuration struct is accepted
    // https://pkg.go.dev/github.com/privacybydesign/irmago/server#Configuration
    // For example:
    "schemes_path": "/path/to/schemes", // optional
    "url": "https://example.com/",      // public URL to this server

    // Attribute IDs in which the login code and client name are to be issued.
    // Must belong to the same credential type, which must exist in a loaded scheme.
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

TLS fields are optional. If they are absent, in production the server must run behind a reverse proxy that handles TLS.

### Running

```sh
uniqueid-issuer /path/to/config.json
```

For a quick local smoke test, use the example config:

```sh
uniqueid-issuer ./conf.example.json
```

You can then start an issuance session against it with the `irma` CLI:

```sh
go install github.com/privacybydesign/irmago/irma@master
irma session --from-package $(curl -X POST -H "Authorization: SecretPresharedToken" http://localhost:1234/session)
```
