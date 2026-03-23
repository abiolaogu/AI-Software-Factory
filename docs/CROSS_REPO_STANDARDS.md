# Cross-Repo Standards

These standards are shared across the StratoGroup AI ecosystem and apply to this repository's contract surface.

## Naming conventions

- Repo slugs use lowercase kebab case, for example `ai-software-factory`.
- Contract artifact ids use `<domain>-<kind>`, for example `software-factory-openapi`.
- Release tags use `contracts/<artifact-id>/v<semver>` for contract artifacts.

## API error model

HTTP and RPC errors should normalize to:

- `code`
- `message`
- `status`
- `category`
- `requestId`
- `retryable`
- `details`

## Auth and session contract

- Authentik is the authentication authority.
- Session claims must carry user id, tenant id, workspace id, roles, permissions, and expiry.
- Frontend and mobile consumers treat backend enforcement as the final authority.

## Tenant and workspace identity

- `tenantId` identifies the customer or internal tenant boundary.
- `workspaceId` identifies the isolated operational scope.
- `region` and `environment` must be explicit where operational routing depends on them.

## Pagination and filtering

- Cursor pagination is preferred for mutable collections.
- Query inputs should use `pageSize`, `cursor`, `sort`, `filter`, and `includeTotal` consistently.

## Observability event naming

- Event names use `bac.<domain>.<resource>.<action>`.
- Metrics and traces should preserve tenant, workspace, region, and request correlation context where safe.

## Design-token naming and export format

- Token exports use nested JSON with stable groups such as `color`, `typography`, `spacing`, `radius`, `elevation`, and `components`.
- Token names must remain stable across web and mobile consumption.

## Route naming and screen ids

- Route ids use dotted domain names such as `software-factory.dashboard`.
- Screen ids use kebab case such as `software-factory-dashboard`.

## Release tagging conventions

- Repo releases use `repo/v<semver>`.
- Contract releases use `contracts/<artifact-id>/v<semver>`.
- Design handoff releases use `handoff/<artifact-set>/v<semver>`.

## Compatibility matrix rules

- Producers publish explicit artifact versions.
- Consumers declare supported semver ranges.
- CI fails on missing artifacts, invalid semver, or unsupported version combinations.
