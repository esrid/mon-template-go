# Authentication

`internal/feature/identity` is a complete vertical feature, not shared platform
code. It owns users, password credentials, Google identities and one-time token
workflows. Session mechanics are delegated to SCS; password hashing is delegated
to `golang.org/x/crypto/bcrypt`; Google token verification is delegated to
`go-oidc` and `x/oauth2`.

## Routes

| Method | Path | Purpose |
|---|---|---|
| POST | `/auth/register` | Create a password user and sign in |
| POST | `/auth/login` | Verify credentials, rotate the session token and sign in |
| POST | `/auth/logout` | Destroy the current session |
| GET | `/auth/me` | Return the current user |
| GET | `/auth/google` | Begin Google OIDC |
| GET | `/auth/google/callback` | Verify Google response and sign in |
| POST | `/auth/email-verification` | Issue a verification token for the signed-in user |
| POST | `/auth/email-verification/confirm` | Consume a verification token |
| POST | `/auth/password-reset` | Accept a reset request without revealing account existence |
| POST | `/auth/password-reset/confirm` | Consume a reset token, replace the password and revoke every session of that user |

## Production work deliberately left to the product

The template does not choose an email provider. The verification endpoint returns
a token and the password-reset endpoint logs it for local development. Replace
both transports before production. Never expose or log these tokens in a deployed
application.

Authorization, roles, tenant membership, account suspension and public-signup
policy are also product decisions. Authentication proves who the user is; it does
not decide what the user may do.

## Google

Google is disabled unless `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, and
`GOOGLE_REDIRECT_URL` are all configured. The callback validates state, nonce,
issuer, audience, expiry and signature through the OIDC library. Only a verified
Google email is accepted.

## Sessions

SCS stores session payloads in SQLite and sends only a random token in an
`HttpOnly`, `SameSite=Lax` cookie. Set `SESSION_SECURE=true` under HTTPS.
Register and login both rotate the session token to prevent fixation, and a
password reset destroys every live session of that user, so a stolen session
does not survive account recovery. Revocation scans live sessions; add a
`user_id` column to the `sessions` table if that scan ever becomes hot.
