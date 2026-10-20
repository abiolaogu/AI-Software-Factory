import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..', '..');
const manifestPaths = [
  'api/openapi/manifest.json',
  'api/proto/manifest.json',
  'api/schemas/manifest.json',
  'compatibility/manifest.json',
  'compatibility/cross-repo-consumer-lock.json',
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

const consumerLock = readJson('compatibility/cross-repo-consumer-lock.json');
assertSemver(consumerLock.schemaVersion, 'cross-repo-consumer-lock schemaVersion');
assert(Array.isArray(consumerLock.contracts) && consumerLock.contracts.length === 5, 'cross-repo-consumer-lock must define 5 boundary contracts');

const consumerByRepo = new Map(
  compatibility.compatibleConsumers.map((entry) => [entry.repo, entry]),
);

for (const contract of consumerLock.contracts) {
  assertSemver(contract.version, `${contract.contractId} version`);
  const schemaFullPath = path.join(root, contract.schemaPath);
  assert(fs.existsSync(schemaFullPath), `Missing boundary schema: ${contract.schemaPath}`);
  const raw = fs.readFileSync(schemaFullPath);
  const digest = crypto.createHash('sha256').update(raw).digest('hex');
  assert(
    digest === contract.sha256,
    `SHA-256 drift for ${contract.contractId}: expected ${contract.sha256}, calculated ${digest}`,
  );
  const producerEntry = consumerByRepo.get(contract.producer);
  assert(producerEntry, `Producer ${contract.producer} for ${contract.contractId} not registered in compatibility/manifest.json`);
  assert(
    Array.isArray(producerEntry.producesContracts) && producerEntry.producesContracts.includes(contract.contractId),
    `Producer ${contract.producer} must declare ${contract.contractId} in producesContracts`,
  );
  for (const consumer of contract.consumers) {
    const consumerEntry = consumerByRepo.get(consumer);
    assert(consumerEntry, `Consumer ${consumer} for ${contract.contractId} not registered in compatibility/manifest.json`);
    assert(
      Array.isArray(consumerEntry.consumesContracts) && consumerEntry.consumesContracts.includes(contract.contractId),
      `Consumer ${consumer} must declare ${contract.contractId} in consumesContracts`,
    );
  }
}

console.log('Integration export and cross-repository consumer lock validation passed.');
