# Compatibility Matrix

| Producer artifact | Version | Compatible consumers |
| --- | --- | --- |
| `software-factory-openapi` | `1.0.0` | `AI-Frontend-Experience@^0.1.0`, `AI-Companion-Mobile@^0.1.0` |
| `software-factory-proto` | `1.0.0` | `AI-Frontend-Experience@^0.1.0` |
| `software-factory-connect-typescript` | `1.0.0` | `AI-Frontend-Experience@^0.1.0` |
| `software-factory-schemas` | `1.0.0` | `AI-Frontend-Experience@^0.1.0`, `AI-Companion-Mobile@^0.1.0` |
| `tenant-context` | `1.0.0` | `Designer2`, `Droplet`, `AppForge`, `SkyForge`, `Nvuto-Forge` |
| `autonomous-mission-request` | `1.0.0` | `AppForge`, `Nvuto-Forge` |
| `design-build-handoff` | `1.0.0` | `AppForge`, `Nvuto-AICO` |
| `build-artifact-handoff` | `1.0.0` | `SkyForge`, `Nvuto-AICO`, `Droplet` |
| `qualification-evidence-envelope` | `1.0.0` | `Nvuto-AICO`, `AppForge`, `SkyForge` |

The machine-readable sources of truth are `compatibility/manifest.json` for published API artifacts and `compatibility/cross-repo-contracts.json` for product-boundary schemas. The contract test suite verifies that every registry entry resolves to a strict Draft 2020-12 schema.
