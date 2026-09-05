# Configuration

Environment variables use the `WOODGATE_` prefix. The example environment and Compose file provide a local PostgreSQL and file-storage configuration.

## Server

| Variable                         | Default or purpose                                         |
| -------------------------------- | ---------------------------------------------------------- |
| `WOODGATE_LISTEN`                | `:8080`                                                    |
| `WOODGATE_URL`                   | Required public origin used by browsers and companion apps |
| `WOODGATE_DATABASE_URL`          | Required PostgreSQL connection URL                         |
| `WOODGATE_LOG_LEVEL`             | `info`; also accepts `debug`, `warn`, or `error`           |
| `WOODGATE_SESSION_COOKIE_SECURE` | `true`; disable only for local HTTP development            |
| `WOODGATE_CORS_ALLOWED_ORIGINS`  | Optional comma-separated browser origins                   |

Browser sessions are stored in PostgreSQL and expire after 14 days. Local users authenticate with stored passwords; `woodgate user create`, `user set-password`, and `user set-roles` administer their access.

Staff roles grant view or edit access to resources. Users can receive roles directly or through directory groups. Directory membership alone does not grant application access. Personal user API keys inherit staff roles. Companion stations use separate pairing secrets and belong to one location.

## Single sign-on

| Variable                      | Default or purpose                                  |
| ----------------------------- | --------------------------------------------------- |
| `WOODGATE_OIDC_ISSUER_URL`    | HTTPS issuer URL                                    |
| `WOODGATE_OIDC_CLIENT_ID`     | Client identifier                                   |
| `WOODGATE_OIDC_CLIENT_SECRET` | Client secret                                       |
| `WOODGATE_OIDC_REDIRECT_URL`  | `WOODGATE_URL` followed by `/api/auth/sso/callback` |
| `WOODGATE_OIDC_SCOPES`        | `openid,email,profile`                              |
| `WOODGATE_OIDC_EMAIL_CLAIM`   | `email`                                             |

Single sign-on is enabled when the issuer, client ID, and client secret are all configured. The email claim must match the directory user's email. Entra synchronization uses the provider's mail address, falling back to its user principal name.

## Directory sync

| Variable                           | Default or purpose          |
| ---------------------------------- | --------------------------- |
| `WOODGATE_ENTRA_TENANT_ID`         | Entra tenant                |
| `WOODGATE_ENTRA_CLIENT_ID`         | Directory client identifier |
| `WOODGATE_ENTRA_CLIENT_SECRET`     | Directory client secret     |
| `WOODGATE_ENTRA_TRANSITIVE_GROUPS` | `false`                     |
| `WOODGATE_ENTRA_SYNC_INTERVAL`     | `1h`                        |

Sync is enabled when all three credentials are configured. Provider object IDs identify directory records across updates; retained check-ins continue to refer to the same people when directory attributes change.

## Storage

Bloby owns uploaded objects. Locations own their background and logo, and check-ins own their photo. Removing or replacing an attachment releases its object when no owner references it.

| Variable                          | Default or purpose                                              |
| --------------------------------- | --------------------------------------------------------------- |
| `WOODGATE_STORAGE_KIND`           | `file`; also accepts `s3`                                       |
| `WOODGATE_STORAGE_FILE_ROOT`      | `data/storage`; persist this directory for file storage         |
| `WOODGATE_STORAGE_CAPABILITY_KEY` | Required for file storage; generate with `openssl rand -hex 32` |
| `WOODGATE_STORAGE_TRANSFER_TTL`   | `15m`                                                           |
| `WOODGATE_STORAGE_S3_BUCKET`      | Required S3 bucket                                              |
| `WOODGATE_STORAGE_S3_REGION`      | Required S3 region                                              |
| `WOODGATE_STORAGE_S3_ENDPOINT`    | Optional HTTPS endpoint for an S3-compatible service            |
| `WOODGATE_STORAGE_S3_ACCESS_KEY`  | Required S3 access key                                          |
| `WOODGATE_STORAGE_S3_SECRET_KEY`  | Required S3 secret key                                          |
| `WOODGATE_STORAGE_S3_PATH_STYLE`  | `false`                                                         |

Persist both PostgreSQL and the configured object storage.

## Reverse proxies

`WOODGATE_HTTP_CLIENT_IP_SOURCE` defaults to `remote_addr`. Alternative sources require an explicit trust boundary:

| Source                | Additional configuration                                                   |
| --------------------- | -------------------------------------------------------------------------- |
| `xff_trusted_cidrs`   | `WOODGATE_HTTP_CLIENT_IP_TRUSTED_CIDRS`, comma-separated CIDRs             |
| `xff_trusted_proxies` | `WOODGATE_HTTP_CLIENT_IP_TRUSTED_PROXY_COUNT`                              |
| `header`              | `WOODGATE_HTTP_CLIENT_IP_HEADER`, set and overwritten by the trusted proxy |
