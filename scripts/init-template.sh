#!/usr/bin/env bash
# Renames the template's placeholders to your project's real values.
#
# Usage:
#   ./scripts/init-template.sh <module_path> <binary_name>
#
# Example:
#   ./scripts/init-template.sh github.com/acme/widget-service widget
#
# This rewrites every occurrence of the placeholder module path
# (github.com/myorg/myapp) and the placeholder binary name (myapp)
# across the working tree, renames the cmd/<binary>/ directory, deletes
# itself when finished, and stages the result for an initial commit.

set -euo pipefail

if [[ $# -ne 2 ]]; then
    echo "usage: $0 <module_path> <binary_name>" >&2
    echo "example: $0 github.com/acme/widget-service widget" >&2
    exit 2
fi

MODULE_PATH="$1"
BINARY_NAME="$2"

if [[ ! "$MODULE_PATH" =~ ^[a-z0-9.-]+/[a-zA-Z0-9._-]+(/[a-zA-Z0-9._-]+)*$ ]]; then
    echo "error: module path '$MODULE_PATH' does not look like a Go module path" >&2
    echo "expected format: hostname/org/repo (e.g. github.com/acme/widget)" >&2
    exit 1
fi

if [[ ! "$BINARY_NAME" =~ ^[a-z][a-z0-9_-]*$ ]]; then
    echo "error: binary name '$BINARY_NAME' must be lowercase, start with a letter, and contain only [a-z0-9_-]" >&2
    exit 1
fi

OLD_MODULE="github.com/myorg/myapp"
OLD_BINARY="myapp"

if [[ "$MODULE_PATH" == "$OLD_MODULE" && "$BINARY_NAME" == "$OLD_BINARY" ]]; then
    echo "error: refusing to rename to the placeholder values" >&2
    exit 1
fi

if ! grep -rq "$OLD_MODULE" . --include='*.go' --include='*.yml' --include='*.yaml' --include='*.md' --include='Dockerfile' 2>/dev/null; then
    echo "error: placeholder '$OLD_MODULE' not found anywhere; has the template already been renamed?" >&2
    exit 1
fi

ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
cd "$ROOT"

echo "==> Rewriting $OLD_MODULE -> $MODULE_PATH"
echo "==> Rewriting $OLD_BINARY -> $BINARY_NAME"

# Rewrite occurrences. Order matters: the module path must be rewritten before
# the binary name, otherwise we'll mangle paths like github.com/myorg/myapp
# into github.com/myorg/<new-binary>.
files_to_rewrite() {
    git ls-files -z 2>/dev/null || \
        find . -type f \
            -not -path './.git/*' \
            -not -path './scripts/init-template.sh' \
            -not -path './node_modules/*' \
            -print0
}

while IFS= read -r -d '' file; do
    case "$file" in
        ./scripts/init-template.sh) continue ;;
        ./.git/*) continue ;;
    esac
    if grep -q "$OLD_MODULE\|$OLD_BINARY" "$file" 2>/dev/null; then
        sed -i "s|$OLD_MODULE|$MODULE_PATH|g" "$file"
        sed -i "s|\\b$OLD_BINARY\\b|$BINARY_NAME|g" "$file"
    fi
done < <(files_to_rewrite)

# Rename cmd/<binary>/ directory.
if [[ -d "cmd/$OLD_BINARY" && "$OLD_BINARY" != "$BINARY_NAME" ]]; then
    echo "==> Renaming cmd/$OLD_BINARY -> cmd/$BINARY_NAME"
    mv "cmd/$OLD_BINARY" "cmd/$BINARY_NAME"
fi

# Refresh go.sum and verify the rename produced a buildable tree.
echo "==> Running go mod tidy"
go mod tidy

echo "==> Verifying build"
go build ./...

echo "==> Removing init script"
rm -- "$0"

echo
echo "Done. Recommended next steps:"
echo "  1) Edit SECURITY.md and replace [REPLACE] markers with your service's actuals."
echo "  2) Edit CLAUDE.md to reflect this project's architecture and conventions."
echo "  3) Edit README.md (replace this template's content with your project's README)."
echo "  4) git add -A && git commit -m 'chore: initial commit from repo_template_go'"
