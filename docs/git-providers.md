# Git provider authentication

Authentication is optional application functionality. It does not change project
manifests, generator templates, repository contents, or generated applications.
No repository creation, cloning, organization lookup, commit, push, or CI/CD API
is implemented by this integration.

## Provider boundary

`internal/gitprovider.Provider` describes a provider ID, instance URL, supported
authentication methods, authentication initiation, account identification,
credential refresh, and disconnection. An authorization session identifies its
method and exposes a cancellable completion contract. Device user codes are
optional method-specific instructions, not a requirement for future providers.
Only GitHub.com device authorization is implemented. GitLab, Bitbucket, and
self-hosted adapters require their own protocols and registrations; the account
schema already distinguishes provider, instance, and stable subject identity.

`application.AccountService` coordinates the provider, metadata repository, and
mockable credential store. Local UI status never assumes SQL metadata means a
valid session. Validation calls the provider once per requested operation and
checks that the credential still belongs to the stored subject ID.

## Register a GitHub OAuth app

1. In GitHub **Settings → Developer settings → OAuth Apps**, register an OAuth app
   you control. Supply your actual application homepage. GitHub's registration
   form also requires a callback URL, but this integration does not use callbacks;
   a loopback placeholder such as `http://127.0.0.1/unused` is not listened on.
2. Enable **Device Flow** in that app's settings.
3. Copy the public **Client ID**. ProvisionHub does not need or accept a client
   secret. Do not copy a client secret into source, settings, or a manifest.
4. Set the client ID before launching ProvisionHub:

```powershell
$env:PROVISIONHUB_GITHUB_CLIENT_ID = "your-registered-public-client-id"
.\provisionhub.exe
```

```sh
PROVISIONHUB_GITHUB_CLIENT_ID="your-registered-public-client-id" ./provisionhub
```

There is no embedded default client ID, fake authorization, or shared secret.
Without configuration, Settings explains how to configure it; generation and
general preferences remain functional. See GitHub's official
[app registration guide](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app)
and [device-flow specification](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow).

## Connect and validate

In Settings → Git Providers, press `c`. ProvisionHub requests a device code and
displays GitHub's verification URL and user code. Press `o` or visit the URL
manually, enter the code, and authorize the app. The code is transient and is not
written to SQLite. Esc cancels the context and polling. Cancellation/expiration
requires starting a new flow; late responses cannot complete an old TUI session.

The request omits OAuth scopes: only public account identification is needed.
It does not request `repo`, private email, organization, or repository-write
permissions. ProvisionHub waits the server's interval before each token request,
continues on `authorization_pending`, adds at least five seconds on `slow_down`,
and handles denial, expiry, timeouts, and cancellation. It then retrieves `/user`
before persisting a connected account. Expiring credentials and refresh material,
if supplied by the registered app, remain in the vault; refresh uses the original
device-flow public-client protocol without a client secret.

HTTP requests have bounded timeouts and response sizes. Redirects are refused;
browser opening accepts only `https://github.com/login/device`. Transport errors
and provider response bodies are not echoed into logs/UI. Secret objects redact
normal Go formatting. Secrets are serialized only for native-vault storage.

## Credential storage and disconnect

The `credentials.Store` interface supports get, replace, and delete. The native
adapter uses `github.com/zalando/go-keyring` with service
`ProvisionHub.git-accounts`:

| Platform | Secure backend | Availability |
| --- | --- | --- |
| Windows | Credential Manager | Current user's native vault |
| macOS | Keychain via the system `/usr/bin/security` utility | May require unlocking/approval; secrets are sent over stdin rather than command arguments |
| Linux | Secret Service over D-Bus | Requires a running session bus and an unlocked compatible keyring, such as GNOME Keyring |

Headless Linux machines may have no usable Secret Service. Native storage errors
are explicit and there is no plaintext-file fallback. General settings and local
generation continue to work. Browser opening uses the OS handler (`rundll32`,
`open`, or `xdg-open`); opening failure leaves manual verification available.
Native vault calls may wait for an OS unlock prompt; cancelling the UI cannot
dismiss an operating-system prompt, but late results are ignored and cancelled
metadata writes trigger cleanup.

Disconnect removes local credentials and account metadata, with an explicit
disconnected marker protecting partial failures. It **does not revoke** the OAuth
grant at GitHub: that API requires additional app credentials and is not invoked.
Revoke the app separately in
[GitHub authorized applications](https://github.com/settings/applications).
See [cross-store recovery guarantees](local-persistence.md#failures-and-recovery).

## Validation

Unit/integration tests use temporary SQLite databases, mocked HTTP endpoints,
and in-memory vault backends. CI requires no live accounts, client IDs, or tokens.
An opt-in native-vault smoke test uses a disposable non-sensitive value; it is
not live OAuth validation. See [development setup](development.md) and
[validation results](settings-validation.md).
