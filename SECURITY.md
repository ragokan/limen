# Security Policy

We take the security of Limen and its users seriously. If you believe you've
found a security vulnerability in Limen, please report it to us privately so we
can investigate and address it before it is publicly disclosed.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues,
discussions, or pull requests.**

Instead, report them by email to:

**[security@limenauth.dev](mailto:security@limenauth.dev)**

To help us triage and respond quickly, please include as much of the following
as you can:

- A description of the issue and its potential impact
- Steps to reproduce, or a proof-of-concept
- The affected version(s), module(s), or plugin(s) (e.g. `limen`,
  `adapters/gorm`, `plugins/credential-password`)
- Any relevant logs, stack traces, or configuration
- Your name / handle if you'd like to be credited in the advisory

Please give us a reasonable window to address the issue before any public
disclosure.

## Scope

In scope:

- The `limen` core library
- First-party adapters under `adapters/`
- First-party plugins under `plugins/`
- Official examples under `examples/` where they demonstrate insecure defaults

Out of scope:

- Vulnerabilities in third-party dependencies (please report those upstream;
  if Limen's usage of a dependency makes the issue exploitable, that _is_ in
  scope)
- Issues that require a compromised host, stolen secret, or attacker-controlled
  build environment
- Missing security hardening that does not correspond to a concrete,
  reproducible vulnerability

## Security Features

Limen's OAuth core validates trusted redirect targets, uses PKCE by default,
supports state storage, and validates OIDC nonce claims for ID-token based
providers. Provider refresh tokens are encrypted at rest by default.

Provider email verification is treated conservatively. OAuth sign-in can create
or use an already-linked provider account even when the provider does not expose
a trusted email-verification signal, but implicit email-based linking to an
existing local user requires a verified provider email.

Magic-link request metadata is stored as magic-link state only by default. It is
not persisted to newly-created user records unless the application explicitly
configures a mapper.

Thank you for helping keep Limen and its users safe.
