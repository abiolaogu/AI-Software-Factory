# Contract Ownership

`AI-Software-Factory` owns the Software Factory backend contract surface.

## Owned

- `software-factory-openapi`
- `software-factory-swagger`
- `software-factory-proto`
- `software-factory-connect-typescript`

## Downstream consumers

- `AI-Frontend-Experience` consumes the OpenAPI bundle, Swagger artifact, and generated Connect-ES types.
- `AI-Companion-Mobile` consumes the OpenAPI bundle and generated Connect-ES types.

## Change policy

- Breaking contract changes require a semver major increment.
- Additive non-breaking changes require a semver minor increment.
- Example or documentation fixes require a semver patch increment when published as contract artifacts.
