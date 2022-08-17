#!/usr/bin/env bash
set -e

git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/equinixmetal".insteadOf "https://github.com/equinixmetal"

exec "$@"
