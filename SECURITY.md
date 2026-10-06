# Security Policy

## Reporting a vulnerability

Please report security issues privately. Do not open a public GitHub issue for sensitive disclosures.

Include:

- Affected version / commit
- Reproduction steps
- Impact assessment

## Security defaults

- Passwords are hashed with bcrypt
- JWT authentication protects API routes
- Registry credentials are encrypted at rest (`ENCRYPTION_KEY`)
- Registry passwords are never returned by the API
- Login endpoints are rate limited
- Destructive actions require confirmation in the UI
- Garbage collection never runs arbitrary user shell input

## Production hardening

Before deploying:

1. Set a strong `JWT_SECRET`
2. Set a strong `ADMIN_PASSWORD`
3. Set a unique `ENCRYPTION_KEY` (16+ characters)
4. Restrict `CORS_ALLOWED_ORIGINS`
5. Disable Swagger (`ENABLE_SWAGGER=false`) unless needed
6. Place RegPilot behind TLS termination
