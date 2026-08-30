# webhook-deploy

A small [Fiber v2](https://gofiber.io) server that receives GitHub `push`
webhooks, verifies the signature, then runs a deploy script in the background.
Supports multiple projects at once (one server, separate secret/script/branch
per project), plus Discord notifications and error logs. A replacement for
cloud GitHub Actions runners — everything runs on your own server, so it's free.

## Architecture

```
cmd/server/main.go        entry point — wires every layer (manual dependency injection)

internal/
├── config/                reads & validates environment variables, including the project list
├── model/                 data structs (DeployRecord, Project, GitHubPushPayload)
├── repository/            data access layer — interface + JSON file implementation
│   └── deploy_repository.go
├── service/
│   ├── github_service.go   HMAC signature verification + GitHub payload parsing
│   ├── deploy_service.go   runs the project script, records the result, sends notifications
│   └── notifier.go         posts to a Discord webhook + writes error logs to file
├── controller/              HTTP layer — takes Fiber requests, calls services, returns JSON
│   └── webhook_controller.go
├── middleware/              custom request logger
└── router/                  route -> handler registration

deploys/                   per-project deploy scripts (one file per project)
└── _examples/              ready-to-use examples: laravel, node, python, docker
```

**Dependency flow**: `controller` → `service` → `repository`. Each layer only
knows the interface of the layer below it, never the concrete implementation.
Swapping storage to SQLite/Postgres later just means writing a new
implementation of `repository.DeployRepository` — `service` and `controller`
stay untouched.

## Multi-project

A single instance can handle many projects/repos, each with its own secret,
deploy script, and target branch. Configured via the `PROJECTS` env var
(comma-separated slugs) plus per-project `<SLUG>_WEBHOOK_SECRET`,
`<SLUG>_DEPLOY_SCRIPT`, `<SLUG>_ALLOWED_BRANCH` — see [Configuration](#configuration).

When a webhook arrives, the server tries the signature against each project
until one matches — so the single `/github` endpoint can be registered on many
GitHub repos at once, just with a different secret per repo.

## Request flow

1. Push to a branch → GitHub sends `POST /github` with the
   `X-Hub-Signature-256` and `X-GitHub-Event` headers.
2. `WebhookController.HandleGitHubWebhook` takes the raw body and matches the
   signature (HMAC-SHA256, `hmac.Equal` to stay safe from timing attacks)
   against every registered project until one matches.
3. If valid and the branch matches (`<SLUG>_ALLOWED_BRANCH`), the controller
   calls `DeployService.TriggerDeploy` — which returns a `deploy_id`
   immediately while the script itself runs in a separate goroutine. This
   matters because GitHub webhook delivery times out after ~10 seconds, while a
   build/deploy can take much longer.
4. The goroutine runs the project script (10 minute timeout), captures
   stdout+stderr, updates the record status (`running` → `success`/`failed`)
   through `DeployRepository.Update`, then sends a Discord notification. On
   failure the error is also written to a log file in `ERROR_LOG_DIR`.
5. Check the result via `GET /deploys`.

## Endpoints

| Method | Path               | Purpose                                        |
| ------ | ------------------ | ---------------------------------------------- |
| POST   | `/github`          | Receive push webhooks from GitHub              |
| GET    | `/deploys?limit=N` | Deploy history across all projects (newest first) |
| GET    | `/health`          | Health check                                   |

## Setup

### 1. Copy the project & install dependencies

```bash
scp -r webhook-deploy/ deploy@server.local:/home/deploy/
ssh deploy@server.local
cd /home/deploy/webhook-deploy
sudo apt install golang-go
go mod tidy
go build -o webhook-deploy-bin ./cmd/server
```

### 2. Write a deploy script per project

Each project needs its own deploy script, looked up by default at
`./deploys/<slug>.deploy.sh` (override with `<SLUG>_DEPLOY_SCRIPT`).
Ready-to-use examples live in [`deploys/_examples/`](deploys/_examples/) —
Laravel, Node, Python, Docker. Copy one and adjust its paths/service names:

```bash
cp deploys/_examples/laravel.deploy.sh deploys/myapp.deploy.sh
chmod +x deploys/myapp.deploy.sh
```

If the script needs `sudo` (e.g. `systemctl restart` / `supervisorctl restart`),
grant passwordless access to that specific command only — not full sudo:

```bash
sudo visudo -f /etc/sudoers.d/webhook-deploy
# contents:
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart your-app
```

### 3. Configure `.env`

`.env` is the single source of truth — both `webhook-deploy.service` (systemd,
via `EnvironmentFile=`) and `webhook-deploy.conf` (supervisor, via a shell
wrapper that sources it) read the same file. The app itself only reads
environment variables; the init system injects them.

```bash
cp .env.example .env
chmod 600 .env   # contains webhook secrets
# edit .env — see Configuration below; replace /<path_project>/ placeholders
```

**Important**: every `<SLUG>_WEBHOOK_SECRET` must be a long random string, not a
guessable password — generate one with `openssl rand -hex 32`.

Then start with systemd:

```bash
sudo cp webhook-deploy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now webhook-deploy
sudo systemctl status webhook-deploy
```

or supervisor:

```bash
sudo cp webhook-deploy.conf /etc/supervisor/conf.d/
sudo supervisorctl reread && sudo supervisorctl update
sudo supervisorctl status webhook-deploy
```

Note: `PROJECTS` and its `<SLUG>_` prefix must match — `PROJECTS=random-project`
expects `RANDOM_PROJECT_WEBHOOK_SECRET`, not `APP_WEBHOOK_SECRET`.

### 4. Expose it to the internet

GitHub needs to reach your server. Pick one:

- A domain/subdomain + reverse proxy (nginx/Caddy) + port forwarding on your router.
- [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/) —
  easiest for a home server, no router ports to open.

### 5. Register the webhook on GitHub

Repo → **Settings → Webhooks → Add webhook**

- Payload URL: `https://your-domain.com/github`
- Content type: `application/json`
- Secret: exactly the same as that project's `<SLUG>_WEBHOOK_SECRET`
- Events: choose **Just the push event**

GitHub sends a `ping` event right away — the server replies
`{"message":"pong","project":"<slug>"}` automatically (see
`HandleGitHubWebhook`, the `X-GitHub-Event` check).

## Configuration

| Env var                 | Required        | Default                      | Purpose                                             |
| ----------------------- | --------------- | ---------------------------- | --------------------------------------------------- |
| `PORT`                  | -               | `9000`                       | HTTP server port                                    |
| `PROJECTS`              | ✓               | - (must be set)              | Comma-separated project slugs, e.g. `PROJECTS=api,web` |
| `<SLUG>_WEBHOOK_SECRET` | ✓ (per project) | -                            | HMAC secret used to verify the GitHub signature     |
| `<SLUG>_DEPLOY_SCRIPT`  | -               | `./deploys/<slug>.deploy.sh` | Path to that project's deploy script                |
| `<SLUG>_ALLOWED_BRANCH` | -               | `refs/heads/main`            | Branch that triggers a deploy                       |
| `DEPLOY_HISTORY_PATH`   | -               | `./deploy_history.json`      | File where deploy history is stored                 |
| `DISCORD_WEBHOOK_URL`   | -               | -                            | If set, sends success/failure notifications to Discord |
| `ERROR_LOG_DIR`         | -               | `./logs`                     | Directory for error logs written on failed deploys  |

`<SLUG>` = the project name from `PROJECTS`, uppercased with `-` replaced by `_`
(e.g. `my-app` → `MY_APP_WEBHOOK_SECRET`).

## Testing locally without GitHub

```bash
export PROJECTS=myapp
export MYAPP_WEBHOOK_SECRET=testsecret123
export MYAPP_DEPLOY_SCRIPT=./deploys/myapp.deploy.sh
./webhook-deploy-bin
```

In another terminal, simulate a push event:

```bash
PAYLOAD='{"ref":"refs/heads/main","after":"abc123","pusher":{"name":"you"},"repository":{"full_name":"you/repo"}}'
SECRET="testsecret123"
SIG="sha256=$(printf '%s' "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')"

curl -X POST http://localhost:9000/github \
  -H "X-GitHub-Event: push" \
  -H "X-Hub-Signature-256: $SIG" \
  -d "$PAYLOAD"

curl http://localhost:9000/deploys
```

## Why it's built this way

- **Raw body for HMAC**: `c.Body()` is used as-is, not re-parsed JSON, because
  GitHub computes the signature over the exact bytes it sent — even a slight
  reserialization breaks the match.
- **Signature matched against every project**: rather than guessing the project
  from a URL/path, the server runs `hmac.Equal` against each registered
  project's secret. Simple, safe from timing attacks, and one `/github`
  endpoint covers every repo.
- **Async deploy (goroutine)**: keeps the response to GitHub fast so a slow
  deploy script doesn't cause a timeout/retry storm.
- **Repository pattern over a JSON file**: plenty for a single-node webhook
  server. If it ever isn't, swap in a SQLite implementation of
  `DeployRepository` without touching `service`/`controller`.
- **Notifier kept out of the deploy logic**: `notifier.Discord` and
  `notifier.AppendError` are plain functions, not interfaces — called directly
  from `deployService`, no abstraction needed beyond that.
- **No `.env` library**: Fiber and uuid are the only external dependencies.
  Config lives in one `.env` file, loaded by the init system (systemd
  `EnvironmentFile=` or a supervisor shell wrapper) — following normal Linux
  service convention.
