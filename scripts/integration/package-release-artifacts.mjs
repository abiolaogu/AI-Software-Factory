import { spawnSync } from 'node:child_process';
import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';

export const repoSlug = 'ai-software-factory';
const root = path.resolve(import.meta.dirname, '..', '..');
const releaseArtifactsDirName = 'release-artifacts/integration';

function ensure(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function readJson(rootDir, relativePath) {
  return JSON.parse(fs.readFileSync(path.join(rootDir, relativePath), 'utf8'));
}

function resolveReleaseTag(explicitTag = process.env.INTEGRATION_RELEASE_TAG ?? process.argv[2]) {
  const releaseTag = explicitTag ?? `${repoSlug}/v0.0.0`;
  ensure(
    new RegExp(`^${repoSlug}/v\\d+\\.\\d+\\.\\d+$`).test(releaseTag),
    `Release tag must match ${repoSlug}/v<semver>, received "${releaseTag}"`,
  );
  return releaseTag;
}

function extractReleaseVersion(releaseTag) {
  return releaseTag.split('/v')[1];
}

function resolveCommitSha(rootDir, explicitCommitSha = process.env.GITHUB_SHA) {
  if (explicitCommitSha) {
    return explicitCommitSha;
  }

  const result = spawnSync('git', ['rev-parse', 'HEAD'], {
    cwd: rootDir,
    encoding: 'utf8',
  });
  ensure(result.status === 0, `Unable to resolve git commit SHA: ${result.stderr}`);
  return result.stdout.trim();
}

function sha256File(filePath) {
  return crypto.createHash('sha256').update(fs.readFileSync(filePath)).digest('hex');
}

function createTarball(rootDir, outputFile, sourcePath) {
  const result = spawnSync('tar', ['-czf', outputFile, sourcePath], {
    cwd: rootDir,
    encoding: 'utf8',
  });
  ensure(result.status === 0, `Failed to create tarball for ${sourcePath}: ${result.stderr}`);
}

function copyFile(rootDir, sourcePath, outputFile) {
  fs.copyFileSync(path.join(rootDir, sourcePath), outputFile);
}

function getArtifactVersion(manifest, artifactId) {
  const artifact = manifest.artifacts.find((candidate) => candidate.artifactId === artifactId);
  ensure(artifact, `Missing artifact ${artifactId} in manifest`);
  return artifact.version;
}

export function buildReleasePlan({
  rootDir = root,
  releaseTag = resolveReleaseTag(),
  commitSha = resolveCommitSha(rootDir),
  publishedAt = new Date().toISOString(),
} = {}) {
  const releaseVersion = extractReleaseVersion(releaseTag);
  const openapiManifest = readJson(rootDir, 'api/openapi/manifest.json');
  const protoManifest = readJson(rootDir, 'api/proto/manifest.json');
  const schemaManifest = readJson(rootDir, 'api/schemas/manifest.json');
  const compatibilityManifest = readJson(rootDir, 'compatibility/manifest.json');

  return {
    schemaVersion: '1.0.0',
    producer: repoSlug,
    releaseTag,
    releaseVersion,
    commitSha,
    publishedAt,
    rootDir,
    releaseArtifactsDir: path.join(rootDir, releaseArtifactsDirName),
    artifacts: [
      {
        artifactId: 'software-factory-openapi',
        kind: 'openapi',
        version: getArtifactVersion(openapiManifest, 'software-factory-openapi'),
        assetName: `${repoSlug}-openapi-${releaseVersion}.tgz`,
        sourcePath: 'api/openapi',
        packaging: 'tar',
      },
      {
        artifactId: 'software-factory-proto',
        kind: 'proto',
        version: getArtifactVersion(protoManifest, 'software-factory-proto'),
        assetName: `${repoSlug}-proto-${releaseVersion}.tgz`,
        sourcePath: 'api/proto',
        packaging: 'tar',
      },
      {
        artifactId: 'software-factory-connect-typescript',
        kind: 'connect-typescript',
        version: getArtifactVersion(protoManifest, 'software-factory-connect-typescript'),
        assetName: `${repoSlug}-connect-typescript-${releaseVersion}.tgz`,
        sourcePath: 'gen/typescript/sdx',
        packaging: 'tar',
      },
      {
        artifactId: 'software-factory-schemas',
        kind: 'schemas',
        version: getArtifactVersion(schemaManifest, 'software-factory-schemas'),
        assetName: `${repoSlug}-schemas-${releaseVersion}.tgz`,
        sourcePath: 'api/schemas',
        packaging: 'tar',
      },
      {
        artifactId: 'software-factory-compatibility',
        kind: 'compatibility',
        version: compatibilityManifest.repoVersion,
        assetName: `${repoSlug}-compatibility-${releaseVersion}.json`,
        sourcePath: 'compatibility/manifest.json',
        packaging: 'copy',
      },
    ],
  };
}

export function packageReleaseArtifacts(options = {}) {
  const plan = buildReleasePlan(options);
  fs.rmSync(plan.releaseArtifactsDir, { force: true, recursive: true });
  fs.mkdirSync(plan.releaseArtifactsDir, { recursive: true });

  const manifestArtifacts = [];
  for (const artifact of plan.artifacts) {
    const outputFile = path.join(plan.releaseArtifactsDir, artifact.assetName);
    if (artifact.packaging === 'tar') {
      createTarball(plan.rootDir, outputFile, artifact.sourcePath);
    } else {
      copyFile(plan.rootDir, artifact.sourcePath, outputFile);
    }

    manifestArtifacts.push({
      artifactId: artifact.artifactId,
      kind: artifact.kind,
      version: artifact.version,
      assetName: artifact.assetName,
      checksumSha256: sha256File(outputFile),
      sourcePath: artifact.sourcePath,
    });
  }

  const manifest = {
    schemaVersion: plan.schemaVersion,
    producer: plan.producer,
    releaseTag: plan.releaseTag,
    commitSha: plan.commitSha,
    publishedAt: plan.publishedAt,
    artifacts: manifestArtifacts,
  };

  const manifestPath = path.join(plan.releaseArtifactsDir, 'integration-release-manifest.json');
  fs.writeFileSync(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);

  return {
    manifest,
    manifestPath,
    outputDir: plan.releaseArtifactsDir,
  };
}

if (import.meta.url === pathToFileURL(process.argv[1]).href) {
  const result = packageReleaseArtifacts();
  console.log(
    JSON.stringify(
      {
        outputDir: path.relative(root, result.outputDir),
        manifestPath: path.relative(root, result.manifestPath),
        artifacts: result.manifest.artifacts.map((artifact) => artifact.assetName),
      },
      null,
      2,
    ),
  );
}
