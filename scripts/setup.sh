#!/usr/bin/env bash
set -euo pipefail

# -----------------------------------------------------------------------------
# setup.sh
#
# This script is just a shortcut. 
# -----------------------------------------------------------------------------

echo "🚀 Installing tools..."
make install-tools

echo "🛠 Setting up development environment..."
make dev-setup

echo "📦 Building the binary..."
make build

echo "✅ Setup complete! Run the app with: ./bin/currentz or make run"
