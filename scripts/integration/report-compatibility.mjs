import fs from 'node:fs';
import path from 'node:path';

const root = path.resolve(import.meta.dirname, '..', '..');
const compatibility = JSON.parse(
  fs.readFileSync(path.join(root, 'compatibility/manifest.json'), 'utf8'),
);

console.log(
  JSON.stringify(
    {
      generatedAt: new Date().toISOString(),
      repo: compatibility.repo,
      repoVersion: compatibility.repoVersion,
      contracts: compatibility.contracts,
      compatibleConsumers: compatibility.compatibleConsumers,
    },
    null,
    2,
  ),
);
