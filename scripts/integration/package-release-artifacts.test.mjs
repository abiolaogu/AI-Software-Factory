import crypto from 'node:crypto';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import assert from 'node:assert/strict';

import { buildReleasePlan, packageReleaseArtifacts, repoSlug } from './package-release-artifacts.mjs';

function writeJson(rootDir, relativePath, value) {
  const fullPath = path.join(rootDir, relativePath);
  fs.mkdirSync(path.dirname(fullPath), { recursive: true });
  fs.writeFileSync(fullPath, `${JSON.stringify(value, null, 2)}\n`);
}

function seedFixtureRepo(rootDir) {
  writeJson(rootDir, 'api/openapi/manifest.json', {
    schemaVersion: '1.0.0',
    artifacts: [{ artifactId: 'software-factory-openapi', version: '1.2.3', path: 'api/openapi/openapi.json' }],
  });
  writeJson(rootDir, 'api/openapi/openapi.json', { openapi: '3.1.0', paths: {} });
  writeJson(rootDir, 'api/proto/manifest.json', {
    schemaVersion: '1.0.0',
    artifacts: [{ artifactId: 'software-factory-proto', version: '2.3.4', path: 'api/proto/sdx.proto' }],
  });
  fs.mkdirSync(path.join(rootDir, 'api/proto'), { recursive: true });
  fs.writeFileSync(path.join(rootDir, 'api/proto/sdx.proto'), 'syntax = "proto3";\n');
  writeJson(rootDir, 'compatibility/manifest.json', {
    schemaVersion: '1.0.0',
    repoVersion: '0.4.0',
  });
}

test('buildReleasePlan derives versioned asset names from the repo release tag', () => {
  const rootDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ai-software-factory-plan-'));
  seedFixtureRepo(rootDir);

  const plan = buildReleasePlan({
    rootDir,
    releaseTag: `${repoSlug}/v0.9.0`,
    commitSha: 'abc1234',
    publishedAt: '2026-03-23T00:00:00.000Z',
  });

  assert.equal(plan.artifacts[0].assetName, 'ai-software-factory-openapi-0.9.0.tgz');
  assert.equal(plan.artifacts[1].assetName, 'ai-software-factory-proto-0.9.0.tgz');
  assert.equal(plan.artifacts[2].assetName, 'ai-software-factory-compatibility-0.9.0.json');
});

test('packageReleaseArtifacts writes checksummed release assets and a machine-readable manifest', () => {
  const rootDir = fs.mkdtempSync(path.join(os.tmpdir(), 'ai-software-factory-package-'));
  seedFixtureRepo(rootDir);

  const { manifest, outputDir } = packageReleaseArtifacts({
    rootDir,
    releaseTag: `${repoSlug}/v1.0.0`,
    commitSha: 'deadbeef',
    publishedAt: '2026-03-23T00:00:00.000Z',
  });

  assert.equal(manifest.releaseTag, `${repoSlug}/v1.0.0`);
  assert.equal(manifest.artifacts.length, 3);
  for (const artifact of manifest.artifacts) {
    const assetPath = path.join(outputDir, artifact.assetName);
    assert.ok(fs.existsSync(assetPath), `expected ${artifact.assetName} to exist`);
    const checksum = crypto.createHash('sha256').update(fs.readFileSync(assetPath)).digest('hex');
    assert.equal(artifact.checksumSha256, checksum);
  }
  assert.ok(fs.existsSync(path.join(outputDir, 'integration-release-manifest.json')));
});
