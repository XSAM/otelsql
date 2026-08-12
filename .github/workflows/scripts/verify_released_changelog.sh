#!/usr/bin/env bash

# Copyright Sam Xie
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

TARGET="${1:?Must provide target ref}"

FILE="CHANGELOG.md"
TEMP_DIR=$(mktemp -d)
echo "Temp folder: $TEMP_DIR"

# Only the latest commit of the feature branch is available
# automatically. To diff with the base branch, we need to
# fetch that too (and we only need its latest commit).
git fetch origin "${TARGET}" --depth=1

# Checkout the previous version on the base branch of the changelog to tmpfolder
git --work-tree="$TEMP_DIR" checkout FETCH_HEAD $FILE

PREVIOUS_FILE="$TEMP_DIR/$FILE"
CURRENT_FILE="$FILE"
PREVIOUS_LOCKED_FILE="$TEMP_DIR/previous_locked_section.md"
CURRENT_LOCKED_FILE="$TEMP_DIR/current_locked_section.md"

# Extract released sections from the previous version
awk '/^<!-- Released section -->/ {flag=1} /^<!-- Released section ended -->/ {flag=0} flag' "$PREVIOUS_FILE" > "$PREVIOUS_LOCKED_FILE"

# Extract released sections from the current version
awk '/^<!-- Released section -->/ {flag=1} /^<!-- Released section ended -->/ {flag=0} flag' "$CURRENT_FILE" > "$CURRENT_LOCKED_FILE"

# Compare the released sections
if ! diff -q "$PREVIOUS_LOCKED_FILE" "$CURRENT_LOCKED_FILE"; then
    echo "Error: The released sections of the changelog file have been modified."
    diff "$PREVIOUS_LOCKED_FILE" "$CURRENT_LOCKED_FILE"
    rm -rf "$TEMP_DIR"
    false
fi

rm -rf "$TEMP_DIR"
echo "The released sections remain unchanged."
