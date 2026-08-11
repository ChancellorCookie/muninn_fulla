#!/bin/bash
set -e
echo "=== Building SvelteKit frontend ==="
cd frontend && npm run build
echo "=== Copying frontend to Go embed dir ==="
cd ..
mkdir -p internal/server/frontend/build
cp -r frontend/.svelte-kit/output/client/* internal/server/frontend/build/
cp frontend/.svelte-kit/output/prerendered/pages/*.html internal/server/frontend/build/
echo "=== Building Go binary ==="
go build -o cmd/fulla/fulla ./cmd/fulla/
echo "=== Done ==="
ls -lh cmd/fulla/fulla
