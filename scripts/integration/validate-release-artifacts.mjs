import fs from 'node:fs';
import path from 'node:path';

import { buildReleasePlan, repoSlug } from './package-release-artifacts.mjs';

const root = path.resolve(import.meta.dirname, '..', '..');
const requiredFiles = [
  '.env.integration.example',
  'api/openapi/manifest.json',
  'api/proto/manifest.json',
  'api/schemas/manifest.json',
  'compatibility/integration-release-manifest.schema.json',
  'compatibility/manifest.json',
];

for (const relativePath of requiredFiles) {
  const fullPath = path.join(root, relativePath);
  if (!fs.existsSync(fullPath)) {
    throw new Error(`Missing release artifact prerequisite: ${relativePath}`);
  }
}

const releaseArtifactsDir = path.join(root, 'release-artifacts', 'integration');
const manifestPath = path.join(releaseArtifactsDir, 'integration-release-manifest.json');

if (fs.existsSync(manifestPath)) {
  const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));
  if (manifest.producer !== repoSlug) {
    throw new Error(`Release manifest producer mismatch: ${manifest.producer}`);
  }

  const plan = buildReleasePlan({
    releaseTag: manifest.releaseTag,
    commitSha: manifest.commitSha,
    publishedAt: manifest.publishedAt,
  });

  for (const expected of plan.artifacts) {
    const assetPath = path.join(releaseArtifactsDir, expected.assetName);
    if (!fs.existsSync(assetPath)) {
      throw new Error(`Missing packaged release asset: ${expected.assetName}`);
    }
  }
}

console.log('Release artifact validation passed.');
