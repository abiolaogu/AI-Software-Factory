import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..', '..');
const manifestPaths = [
  'api/openapi/manifest.json',
  'api/proto/manifest.json',
  'api/schemas/manifest.json',
  'compatibility/manifest.json',
];

function readJson(relativePath) {
  const fullPath = path.join(root, relativePath);
  return JSON.parse(fs.readFileSync(fullPath, 'utf8'));
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function assertSemver(value, label) {
  assert(/^\d+\.\d+\.\d+$/.test(value), `${label} must be semver, received "${value}"`);
}

for (const manifestPath of manifestPaths) {
  assert(fs.existsSync(path.join(root, manifestPath)), `Missing required manifest: ${manifestPath}`);
}

for (const manifestPath of ['api/openapi/manifest.json', 'api/proto/manifest.json', 'api/schemas/manifest.json']) {
  const manifest = readJson(manifestPath);
  assert(Array.isArray(manifest.artifacts) && manifest.artifacts.length > 0, `${manifestPath} must define artifacts`);
  for (const artifact of manifest.artifacts) {
    assertSemver(artifact.version, `${artifact.artifactId} version`);
    assert(fs.existsSync(path.join(root, artifact.path)), `Artifact path does not exist: ${artifact.path}`);
    if (artifact.sourcePath) {
      assert(fs.existsSync(path.join(root, artifact.sourcePath)), `Artifact source path does not exist: ${artifact.sourcePath}`);
    }
  }
}

const compatibility = readJson('compatibility/manifest.json');
assertSemver(compatibility.repoVersion, 'repoVersion');
for (const contract of compatibility.contracts) {
  assertSemver(contract.version, `${contract.artifactId} compatibility version`);
}

console.log('Integration export validation passed.');
