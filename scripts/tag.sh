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
    
    OLD_VERSION="${OLD_TAG#v}"
    NEW_VERSION="${NEW_TAG#v}"
    
    # Find files containing either the tag or the version without the v
    files_with_v=$(git grep -l "$OLD_TAG" | grep -v "website/components/benchmarks.templ" | grep -v "tests/benchmark.txt" | grep -v "scripts/tag.sh" || true)
    files_without_v=$(git grep -l "$OLD_VERSION" | grep -v "website/components/benchmarks.templ" | grep -v "tests/benchmark.txt" | grep -v "scripts/tag.sh" || true)
    
    all_files=$(echo -e "$files_with_v\n$files_without_v" | sort | uniq | grep -v '^$')
    
    if [ -n "$all_files" ]; then
        echo "Updating references in:"
        echo "$all_files"
        for f in $all_files; do
            # Replace tag with tag (e.g. v0.0.1 -> v0.0.2)
            sed -i "s/$OLD_TAG/$NEW_TAG/g" "$f"
            # Replace version with version (e.g. 0.0.1 -> 0.0.2)
            sed -i "s/$OLD_VERSION/$NEW_VERSION/g" "$f"
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
