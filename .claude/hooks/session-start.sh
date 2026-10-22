#!/bin/bash
# Installs the OpenSpec CLI that the /opsx:* commands and openspec-* skills in .claude/ call.
# Remote (Claude Code on the web) sessions only: a local machine installs it once by hand.
# Pinned to the version that generated .claude/; bump it together with `openspec update`.
set -euo pipefail

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

OPENSPEC_VERSION="1.14.3"

if command -v openspec >/dev/null 2>&1 && [ "$(openspec --version 2>/dev/null)" = "$OPENSPEC_VERSION" ]; then
  exit 0
fi

npm install -g "@fission-ai/openspec@${OPENSPEC_VERSION}" --no-fund --no-audit
