#!/usr/bin/env bash
# Irancell Omnichannel — run every quality gate (backend + frontend).
# Prereqs: go (1.26+), node (22+) and npm on PATH.
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "== Backend: gofmt =="
cd "$ROOT/backend"; test -z "$(gofmt -l .)"
echo "== Backend: vet =="; go vet ./...
echo "== Backend: build =="; go build ./...
echo "== Backend: test -count=1 =="; go test -count=1 ./...

echo "== Frontend: eslint =="
cd "$ROOT/frontend"; npm run lint
echo "== Frontend: tsc --noEmit =="; npx tsc --noEmit
echo "== Frontend: vitest =="; npx vitest run
echo "== Frontend: next build =="; npm run build

echo "ALL GATES GREEN"
