# workflow-plugin-scalekit

Scalekit enterprise SSO and SCIM Directory Sync provider plugin for Workflow.

This plugin uses the official Scalekit Go SDK
`github.com/scalekit-inc/scalekit-sdk-go/v2` and exposes only the management
operations that are backed by that SDK.

## Capabilities

- `scalekit.provider` module configured with Scalekit environment URL, client
  ID, and client secret.
- Auth provider descriptor step for dynamic Workflow admin catalog integration.
- SSO connection create/read/list/enable/disable/delete steps.
- Directory create/read/list/enable/disable/delete steps.
- Directory user and group list steps.

The descriptor advertises `enterprise_sso` and `directory_sync` only. Identity
management, OAuth server, MFA, and authorization capabilities belong to other
providers.

## Security

Keep Scalekit client secrets in a Workflow secret source. Run these management
steps server-side through an admin surface protected by Workflow auth/authz
scopes, and grant the Scalekit credential only the permissions needed by the
enabled SSO and Directory Sync operations.

## Install

```sh
wfctl plugin install workflow-plugin-scalekit
```

## License

MIT
