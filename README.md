# AppSuite

`github.com/fastygo/app-suite` is the official multi-profile proof assembly for
FastyGo Platform.

It proves that monolith-like apps and multi-workspace apps use the same runtime:

```text
profile.Profile -> modulehost.Assemble(profile) -> workspace registry -> routes
```

There is no `monolith/spaces` runtime switch.

## Commands

```bash
bun verify
bun go
```

The server listens on `127.0.0.1:8080` by default. Set `APPSUITE_PROFILE` to
choose a profile:

```bash
APPSUITE_PROFILE=crm-leads bun go
```

## Profiles

- `gocms-admin`: root CMS admin at `/go-admin`, API at `/go-json`.
- `crm-leads`: standalone CRM at `/go-admin`, API at `/go-json`.
- `gocms-workspaces-full`: website at `/`, root admin at `/go-admin`, root API at `/go-json`, spaces under `/go-admin/spaces/{space}` and `/go-json/spaces/{space}`.
- `headless`: API-first CMS profile.
- `local-offline`: CRM profile with local/offline metadata.
- `demo-suite`: CMS, CRM, and monitoring demo profile.

JSON profile fixtures live in `profiles/`.

## Workspace Routes

Root admin:

```text
/go-admin
/go-json
```

Workspace hub:

```text
/go-admin/spaces
```

Additional spaces:

```text
/go-admin/spaces/sales
/go-json/spaces/sales
```

The root admin is not a task space. Task spaces appear only under
`/go-admin/spaces/{space}` and `/go-json/spaces/{space}`.

## Shared Module Plans

Shared contacts and shared activity timeline plans are documented in
`docs/shared-modules.md`. They are intentionally not implemented yet; ModuleCRM
owns contacts and activities until real duplication appears across multiple
product modules.
