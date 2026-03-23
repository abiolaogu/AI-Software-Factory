# Contract Ownership

`AI-Software-Factory` owns the Software Factory backend contract surface.

## Owned

- `software-factory-openapi`
- `software-factory-proto`
- `software-factory-connect-typescript`
- `software-factory-schemas`

## Downstream consumers

- `AI-Frontend-Experience` consumes the OpenAPI bundle, proto bundle, and generated Connect-ES types.
- `AI-Companion-Mobile` consumes the OpenAPI bundle.

## Change policy

- Breaking contract changes require a semver major increment.
- Additive non-breaking changes require a semver minor increment.
- Example or documentation fixes require a semver patch increment when published as contract artifacts.
