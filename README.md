# AppSuite

`github.com/fastygo/app-suite` is the official multi-profile proof assembly for
FastyGo Platform.

It proves that monolith-like apps and multi-workspace apps use the same runtime:

```text
profile.Profile -> modulehost.Assemble(profile) -> workspace registry -> routes
```

There is no `monolith/spaces` runtime switch.

AppSuite is the launcher assembly. It imports reusable app bundles from
`github.com/fastygo/app-gocms/pkg/app` and `github.com/fastygo/app-crm/pkg/app`,
mounts AppCMS in the root workspace, and mounts AppCRM-style products as spaces.
CMS and CRM templates stay with their source apps; AppSuite owns only launcher
chrome such as the workspace switcher and space directory.

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
- `demo-suite`: CMS, CRM, monitoring, support, and chat demo modules.
- `optional-remote-services`: fixture profile for optional remote spaces.

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

## Remote Space Fixture

AppSuite tests prove that an optional remote module can be added to the same
workspace registry as compiled-in modules. The remote support fixture mounts at:

```text
/go-admin/spaces/remote-support
/go-json/spaces/remote-support
```

The default runtime still uses bundled CMS, CRM, and monitoring modules.
