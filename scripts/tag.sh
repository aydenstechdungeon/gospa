#!/bin/bash
set -e

NEW_TAG=$1

if [ -z "$NEW_TAG" ]; then
    echo "Usage: $0 <tag>"
    exit 1
fi

# Get module name from root go.mod
MODULE_NAME=$(head -1 go.mod | cut -d' ' -f2)
if [ -z "$MODULE_NAME" ]; then
    echo "Error: Could not determine module name from go.mod"
    exit 1
fi

echo "Detected module: $MODULE_NAME"

# Get the latest tag
OLD_TAG=$(git tag --sort=-v:refname | head -n 1)

# If there is a new tag and it's different from the old one, we bump
if [ -n "$OLD_TAG" ] && [ "$NEW_TAG" != "$OLD_TAG" ]; then
    echo "Bumping version from $OLD_TAG to $NEW_TAG..."

    OLD_VERSION="${OLD_TAG#v}"
    NEW_VERSION="${NEW_TAG#v}"

    # Find go.mod files that reference this module
    echo "Searching for module references..."

    # Find all go.mod files in the repo (excluding vendor)
    go_mod_files=$(find . -name "go.mod" -not -path "./vendor/*" -not -path ".git/*" | sed 's|^\./||')

    updated_dirs=""

    for mod_file in $go_mod_files; do
        dir=$(dirname "$mod_file")

        # Check if this go.mod references our module
        if grep -q "$MODULE_NAME v$OLD_VERSION" "$mod_file" 2>/dev/null || \
           grep -q "$MODULE_NAME $OLD_TAG" "$mod_file" 2>/dev/null; then

            echo "Updating $mod_file..."

            # Update the module version in go.mod
            # Match: github.com/aydenstechdungeon/gospa v0.1.14
            sed -i "s|$MODULE_NAME v$OLD_VERSION|$MODULE_NAME v$NEW_VERSION|g" "$mod_file"

            # Also check for tag format (less common in go.mod but possible)
            sed -i "s|$MODULE_NAME $OLD_TAG|$MODULE_NAME $NEW_TAG|g" "$mod_file"

            updated_dirs="$updated_dirs $dir"
        fi
    done

    # Find other files that reference the tag (documentation, README, etc.)
    # Exclude go.mod/go.sum files now handled separately, and binary files
    other_files=$(git grep -l "$OLD_TAG" -- "*.md" "*.templ" "*.go" ":!vendor/" ":!*.mod" ":!*.sum" 2>/dev/null || true)

    if [ -n "$other_files" ]; then
        echo "Updating references in other files..."
        echo "$other_files" | while read -r f; do
            # Only update lines that don't look like external dependency references
            # Skip lines that look like: github.com/other/package v0.1.14
            sed -i "/github\.com\/.*\/.* v/s/$OLD_TAG/$NEW_TAG/g" "$f"
            sed -i "/github\.com\/.*\/.* v/s/$OLD_VERSION/$NEW_VERSION/g" "$f"
        done
    fi

    # Regenerate go.sum files by running go mod tidy in each updated directory
    if [ -n "$updated_dirs" ]; then
        echo ""
        echo "Regenerating go.sum files..."
        for dir in $updated_dirs; do
            if [ -f "$dir/go.mod" ]; then
                echo "  Running go mod tidy in $dir..."
                (cd "$dir" && go mod tidy 2>/dev/null || echo "    Warning: go mod tidy failed in $dir (may need manual check)")
            fi
        done
    fi

    # Check for any remaining go.sum files that might reference the old version
    # and run go mod tidy on them too
    go_sum_files=$(find . -name "go.sum" -not -path "./vendor/*" -not -path ".git/*" | sed 's|^\./||')
    for sum_file in $go_sum_files; do
        dir=$(dirname "$sum_file")
        if grep -q "$MODULE_NAME $OLD_VERSION" "$sum_file" 2>/dev/null || \
           grep -q "$MODULE_NAME $OLD_TAG" "$sum_file" 2>/dev/null; then
            echo "  Cleaning up $sum_file..."
            (cd "$dir" && go mod tidy 2>/dev/null || true)
        fi
    done

    # Commit all changes
    if [ -n "$updated_dirs" ] || [ -n "$other_files" ]; then
        git add -A
        if git diff --cached --quiet; then
            echo "No changes to commit"
        else
            git commit -m "chore: bump version to $NEW_TAG" || echo "Commit failed or nothing to commit"
        fi
    fi
fi

# Get current branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo ""
echo "Tagging $NEW_TAG and pushing to $BRANCH..."

# Force update the tag to current HEAD
git tag -f "$NEW_TAG"

# Push the current branch and the tag
git push origin "$BRANCH"
git push -f origin "$NEW_TAG"

echo ""
echo "Successfully updated tag $NEW_TAG"
