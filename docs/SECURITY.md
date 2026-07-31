# Security Guide

## Scope

The repository provides a secure operational baseline, not a complete
application security system.

Already present:

- bounded HTTP server timeouts and header size;
- graceful shutdown;
- request IDs and structured access logging;
- panic recovery;
- `X-Content-Type-Options: nosniff`;
- a strict referrer policy;
- parameterized SQL in feature adapters;
- SQLite foreign keys and connection policy;
- non-root production container;
- race tests, lint, vet, and `govulncheck` in CI.

Intentionally not selected by the template:

- authentication and authorization;
- sessions and cookie policy;
- CSRF protection;
- CORS policy;
- content security policy;
- rate limiting;
- secret manager;
- multi-tenancy model;
- encryption key management;
- audit event storage.

Every real product must select these based on its threat model.

## Threat-model first

Before adding sensitive functionality, document:

- assets being protected;
- actors and trust boundaries;
- public vs authenticated routes;
- roles/permissions;
- personal or regulated data;
- external callbacks/webhooks;
- tenant boundaries;
- availability and abuse risks;
- backup and recovery requirements.

Security mechanisms without a threat model tend to protect the wrong boundary.

## Input handling

Treat every external value as untrusted:

- path variables;
- query values;
- headers;
- cookies;
- JSON/form bodies;
- uploaded files;
- webhook payloads;
- environment variables;
- data returned by external providers.

Apply limits before expensive processing:

- body size;
- header size;
- upload size/count;
- string and collection lengths;
- pagination maximums;
- timeout and concurrency limits.

Transport validation proves the representation is readable. Feature validation
proves the operation is valid.

## SQL and SQLite

- Use placeholders for all data values.
- Never concatenate user input into SQL identifiers or clauses.
- Whitelist any dynamic sort field or direction.
- Enable and preserve foreign keys.
- Use database constraints for uniqueness and referential integrity.
- Translate constraint errors into neutral feature errors.
- Do not expose SQL text, driver codes, DSNs, or schema details to clients.
- Keep transactions short and cover complete atomic invariants.

SQLite is a file database. Protect the database and backup files with filesystem
permissions appropriate to the deployment user.

## Authentication

When a project adds authentication:

- hash passwords with a current password hashing algorithm designed for
  passwords (for example Argon2id or bcrypt with an appropriate cost);
- never use SHA-family fast hashes directly for password storage;
- rate-limit authentication attempts;
- avoid revealing whether an account exists where enumeration matters;
- support credential rotation and recovery securely;
- invalidate or rotate sessions after login, privilege change, password change,
  and account recovery;
- require re-authentication for high-risk actions;
- consider MFA for administrative access.

Authentication verifies identity. Authorization must still occur for each
protected operation.

## Sessions and cookies

For browser sessions:

- generate cryptographically random opaque session IDs;
- store only the minimum required data in cookies;
- set `HttpOnly`;
- set `Secure` in HTTPS deployments;
- set `SameSite=Lax` or stricter unless the flow requires otherwise;
- use a narrow cookie path/domain;
- rotate IDs after authentication state changes;
- enforce idle and absolute expiration;
- revoke server-side sessions when appropriate;
- never log session cookie values.

Do not place authorization decisions solely in client-side tokens without a
well-reviewed token and revocation design.

## CSRF

Browser-authenticated state-changing routes require CSRF protection even when
cookies use SameSite.

Use one coherent strategy:

- synchronizer token;
- double-submit cookie with robust binding;
- framework/library mechanism appropriate to the project.

Also verify method, content type, and Origin/Referer where appropriate. Do not
use GET for state changes.

## CORS

Do not enable permissive CORS by default.

When a separate frontend needs cross-origin access:

- allow exact trusted origins;
- allow only required methods and headers;
- do not combine wildcard origins with credentials;
- validate preflight behavior;
- keep CORS separate from authentication and authorization.

CORS is a browser policy, not an API access control mechanism.

## Browser output, XSS, and CSP

- Render HTML through an escaping template system.
- Never mark untrusted strings as trusted HTML.
- Avoid injecting data into inline JavaScript.
- Encode for the actual output context.
- Sanitize user-authored rich HTML with a maintained allowlist sanitizer.
- Set a project-specific Content Security Policy.
- Prefer external scripts/styles or nonce/hash-based policies over `unsafe-inline`.
- Set frame-ancestor policy through CSP when clickjacking matters.

The base middleware intentionally does not guess a CSP because the correct policy
depends on the frontend and third-party resources.

## Authorization

Authorization belongs at the server-side feature boundary.

- Deny by default.
- Check both the action and the target resource.
- Do not trust hidden form fields, route ownership, or UI visibility.
- Keep privileged operations explicit.
- Test horizontal and vertical privilege escalation.
- Return neutral errors when resource existence is sensitive.

## Multi-tenancy

When adding multi-tenancy:

1. Make tenant identity explicit and authenticated.
2. Include `tenant_id` in every tenant-owned read and write.
3. Include tenant scope in unique constraints and foreign keys where applicable.
4. Never query globally then filter in Go.
5. Prevent optional tenant parameters for tenant-owned operations.
6. Include tenant scope in caches, jobs, idempotency keys, object paths, and
   external provider metadata.
7. Test that tenant A cannot read, update, delete, or infer tenant B's records.
8. Use transactions/locking for cross-row tenant invariants.
9. Avoid trusting a caller-supplied tenant ID without comparing it to the
   authenticated identity.

Tenant isolation is a data-boundary design, not merely middleware.

## Webhooks and external callbacks

- Verify provider signatures before parsing trusted fields.
- Use the raw request bytes required by the provider's signature algorithm.
- Enforce timestamp/replay windows where supported.
- Deduplicate events with provider event IDs.
- Store/process events idempotently.
- Restrict body size and content type.
- Do not trust source IP alone unless the provider guarantees and documents it.
- Return quickly and process slow work asynchronously when required.
- Never log secrets or full sensitive payloads.

## Secrets and configuration

- Keep secrets out of source control and container images.
- Use deployment secret injection or a secret manager.
- Separate development and production credentials.
- Rotate credentials and design for overlap during rotation.
- Avoid including secret values in wrapped errors.
- Restrict production database file and volume access.
- Do not use `.env` files as a production secret-management strategy.

## Logging and privacy

Log enough to operate the service, not enough to reconstruct private user data.

Never log:

- passwords;
- authorization headers;
- session/token values;
- payment credentials;
- full identity documents;
- unnecessary message/body contents.

Use identifiers and request IDs where possible. Establish retention and access
controls for logs containing personal data.

## Rate limiting and abuse

Add endpoint-specific controls for:

- login and recovery;
- account creation;
- expensive search/export;
- outbound email/SMS/voice;
- uploads;
- public APIs and webhooks.

Rate limits must be paired with timeouts, body limits, quotas, and economic abuse
controls. Avoid unbounded goroutines or queues.

## File uploads

When uploads are introduced:

- enforce size and count limits;
- generate server-side object names;
- do not trust the filename or declared MIME type;
- inspect content signatures where practical;
- store outside executable/static roots unless explicitly intended;
- prevent path traversal;
- scan risky formats when required;
- serve downloads with safe content disposition and content type;
- authorize every download.

## Dependencies and supply chain

- Keep `go.sum` committed.
- Review new direct dependencies.
- Pin CI actions to maintained major versions or immutable commits according to
  organizational policy.
- Run `govulncheck` regularly.
- Keep Go and the Alpine runtime image patched.
- Regenerate images after security updates.
- Do not execute untrusted build scripts in CI.

## Container and deployment

The production image runs as a non-root `app` user; preserve that property.

- Mount writable data only where required.
- Keep filesystem permissions narrow.
- Use TLS at a trusted reverse proxy or in the application.
- Restrict network exposure for administrative endpoints.
- Configure resource limits.
- Keep readiness and liveness semantics distinct.
- Test graceful termination under real orchestration.

## SQLite backup and recovery

A live SQLite database may use WAL and shared-memory files. Do not assume that
copying only `app.db` at an arbitrary instant is a valid backup.

Use a safe strategy such as:

- SQLite online backup API;
- `VACUUM INTO` where appropriate;
- a controlled checkpoint/snapshot process;
- filesystem/storage snapshots with documented consistency guarantees.

Encrypt backups where required, restrict access, test restoration, and define
retention.

## Security review checklist

Before release:

- [ ] Trust boundaries and sensitive data are documented.
- [ ] All protected routes authenticate and authorize.
- [ ] Browser state changes have CSRF protection.
- [ ] Cookies use secure attributes.
- [ ] CORS is absent or allowlisted.
- [ ] Input/body/upload limits exist.
- [ ] SQL is parameterized and constraints enforce invariants.
- [ ] Cross-tenant tests pass where tenancy exists.
- [ ] Webhook signatures and replay protection are tested.
- [ ] Logs exclude secrets and unnecessary personal data.
- [ ] Secrets are injected outside the repository/image.
- [ ] Dependencies and images have been vulnerability-checked.
- [ ] Backup and restore have been tested.
- [ ] Production uses HTTPS.
