# Cross-Repo Release Flow

1. Update the Software Factory API contract or generated artifacts.
2. Increment the affected contract artifact version in the manifest files when compatibility changes.
3. Tag the producer release with `ai-software-factory/v<semver>`.
4. Producer CI validates OpenAPI, proto, docs, and release metadata, then packages the release assets into GitHub Releases.
5. The uploaded release contains `integration-release-manifest.json`, the versioned OpenAPI tarball, the versioned proto tarball, and the compatibility manifest asset.
6. Downstream frontend and mobile repos resolve the selected release from GitHub Releases, verify checksums, regenerate client bindings, and fail CI if compatibility ranges no longer hold.
