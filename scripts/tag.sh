#!/bin/bash
set -e

NEW_TAG=$1

if [ -z "$NEW_TAG" ]; then
    echo "Usage: $0 <tag>"
    exit 1
fi

# Get the latest tag
OLD_TAG=$(git tag --sort=-v:refname | head -n 1)

# If there is a new tag and it's different from the old one, we bump
if [ -n "$OLD_TAG" ] && [ "$NEW_TAG" != "$OLD_TAG" ]; then
    echo "Bumping version from $OLD_TAG to $NEW_TAG..."
    
    # Find files containing the old tag, excluding specific ones
    # We also exclude .git directory implicitly by using git grep
    files=$(git grep -l "$OLD_TAG" | grep -v "website/components/benchmarks.templ" | grep -v "tests/benchmark.txt" | grep -v "scripts/tag.sh" || true)
    
    if [ -n "$files" ]; then
        echo "Updating references in:"
        echo "$files"
        for f in $files; do
            # Use a different delimiter for sed just in case but / should be fine for tags
            sed -i "s/$OLD_TAG/$NEW_TAG/g" "$f"
        done
        
        git add .
        git commit -m "chore: bump version to $NEW_TAG" || echo "No changes to commit"
    fi
fi

# Get current branch
BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "Tagging $NEW_TAG and pushing to $BRANCH..."

# Force update the tag to current HEAD
git tag -f "$NEW_TAG"

# Push the current branch and the tag
git push origin "$BRANCH"
git push -f origin "$NEW_TAG"

echo "Successfully updated tag $NEW_TAG"
