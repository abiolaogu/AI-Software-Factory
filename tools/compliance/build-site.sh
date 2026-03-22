#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
site_dir="${repo_root}/dist/docs"

rm -rf "${site_dir}"
mkdir -p "${site_dir}/swagger" "${site_dir}/docs"

cp "${repo_root}/api/openapi/openapi.json" "${site_dir}/openapi.json"
cp "${repo_root}/api/openapi/openapi.yaml" "${site_dir}/openapi.yaml"

cat >"${site_dir}/index.html" <<'EOF'
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="0; url=./swagger/">
  <title>NeuroForge SDX API Docs</title>
</head>
<body>
  <a href="./swagger/">Open Swagger UI</a>
</body>
</html>
EOF

cat >"${site_dir}/swagger/index.html" <<'EOF'
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>NeuroForge SDX Swagger UI</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '../openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis]
    });
  </script>
</body>
</html>
EOF

cat >"${site_dir}/docs/index.html" <<'EOF'
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>NeuroForge SDX ReDoc</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>body { margin: 0; }</style>
</head>
<body>
  <redoc spec-url="../openapi.json"></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>
EOF

echo "documentation site built at ${site_dir}"
