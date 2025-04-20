#!/usr/bin/env bash

set -e -o pipefail

type gh > /dev/null 2>&1 || { echo >&2 'ERROR: Github CLI is required (see https://cli.github.com)'; exit 1; }
type go-coverage-report > /dev/null 2>&1 || { echo >&2 'ERROR: Script requires "go-coverage-report" binary in PATH'; exit 1; }

setup_env_variables()  {
  GITHUB_BASELINE_WORKFLOW=${GITHUB_BASELINE_WORKFLOW:-CI}
  TARGET_BRANCH=${TARGET_BRANCH:-main}
  COVERAGE_ARTIFACT_NAME=${COVERAGE_ARTIFACT_NAME:-code-coverage}
  COVERAGE_FILE_NAME=${COVERAGE_FILE_NAME:-coverage.txt}
  SKIP_COMMENT=${SKIP_COMMENT:-false}
  COMMENT_TAG="<-- ${COMMENT_TAG:-Go Coverage Report} -->"
  CONFIG_PATH="${CONFIG_PATH:-}"

  OLD_COVERAGE_PATH=.github/outputs/old-coverage.txt
  NEW_COVERAGE_PATH=.github/outputs/new-coverage.txt
  COVERAGE_COMMENT_PATH=.github/outputs/coverage-comment.md
  CHANGED_FILES_PATH=.github/outputs/all_modified_files.json

  if [[ -z ${GITHUB_REPOSITORY+x} ]]; then
      echo "Missing GITHUB_REPOSITORY environment variable"
      exit 1
  fi

  if [[ -z ${GITHUB_TOKEN+x} ]]; then
      echo "Missing GITHUB_TOKEN environment variable"
      exit 1
  fi

  if [[ -z ${GITHUB_PULL_REQUEST_NUMBER+x} ]]; then
      echo "Missing GITHUB_PULL_REQUEST_NUMBER environment variable"
      exit 1
  fi

  if [[ -z ${GITHUB_RUN_ID+x} ]]; then
      echo "Missing GITHUB_RUN_ID environment variable"
      exit 1
  fi

  if [[ -z ${GITHUB_OUTPUT+x} ]]; then
      echo "Missing GITHUB_OUTPUT environment variable"
      exit 1
  fi

  # If GITHUB_BASELINE_WORKFLOW_REF is defined, extract the workflow file path from it and use it instead of GITHUB_BASELINE_WORKFLOW
  if [[ -n ${GITHUB_BASELINE_WORKFLOW_REF+x} ]]; then
      GITHUB_BASELINE_WORKFLOW=$(basename "${GITHUB_BASELINE_WORKFLOW_REF%%@*}")
  fi

  export GH_REPO="$GITHUB_REPOSITORY"
  export GH_TOKEN="$GITHUB_TOKEN"
}

start_group() {
    echo "::group::$*"
    { set -x; return; } 2>/dev/null
}

end_group() {
    { set +x; return; } 2>/dev/null
}

check_coverage_artifact() {
  local run_id="$1"
  local artifact_name="$2"

  # Get artifacts for the given workflow run
  ARTIFACTS=$(gh api repos/"$GITHUB_REPOSITORY"/actions/runs/"$run_id"/artifacts --jq '.artifacts[].name')

  # Check if the artifact exists
  if echo "$ARTIFACTS" | grep -q "^$artifact_name$"; then
    return 0
  else
    return 1
  fi
}

download_coverage_artifact() {
  local run_id="$1"
  local artifact_name="$2"
  local file_name="$3"
  local move_to_path="$4"


  gh run download "$run_id" --name="$artifact_name" --dir="/tmp/gh-run-download-$run_id"
  mv "/tmp/gh-run-download-$run_id/$file_name" "$move_to_path"
  rm -r "/tmp/gh-run-download-$run_id"
}

post_comment() {
  local pr_number="$1"
  local body="$2"
  local tag="$3"

  COMMENT_ID=$(gh api "repos/${GITHUB_REPOSITORY}/issues/${GITHUB_PULL_REQUEST_NUMBER}/comments" -q '.[] | select(.user.login=="github-actions[bot]" and (.body | test("'"$tag"'")) ) | .id' | head -n 1)

  if [ -z "$COMMENT_ID" ]; then
    echo "Creating new coverage report comment"
  else
    echo "Replacing old coverage report comment"
    gh api -X DELETE "repos/${GITHUB_REPOSITORY}/issues/comments/${COMMENT_ID}"
  fi

  gh pr comment "$pr_number" --body-file="$body"
}

main() {
  setup_env_variables

  LAST_SUCCESSFUL_RUN_ID=$(gh run list --status=success --branch="$TARGET_BRANCH" --workflow="$GITHUB_BASELINE_WORKFLOW" --event=push --json=databaseId --limit=1 -q '.[] | .databaseId')
  if [ -z "$LAST_SUCCESSFUL_RUN_ID" ]; then
    echo "No successful run found on the target branch"
    exit 0
  fi

  if ! check_coverage_artifact "$LAST_SUCCESSFUL_RUN_ID" "$COVERAGE_ARTIFACT_NAME"; then
    echo "No Artifact $COVERAGE_ARTIFACT_NAME found on the target branch"
    exit 0
  fi

  start_group "Download code coverage results from current run"
  download_coverage_artifact "$GITHUB_RUN_ID" "$COVERAGE_ARTIFACT_NAME" "$COVERAGE_FILE_NAME" "$NEW_COVERAGE_PATH"
  end_group

  start_group "Download code coverage results from target branch"
  download_coverage_artifact "$LAST_SUCCESSFUL_RUN_ID" "$COVERAGE_ARTIFACT_NAME" "$COVERAGE_FILE_NAME" "$OLD_COVERAGE_PATH"
  end_group

  start_group "Compare code coverage results"
  REPORT=$(go-coverage-report \
      -root="$ROOT_PACKAGE" \
      -trim="$TRIM_PACKAGE" \
      -config="$CONFIG_PATH" \
      "$OLD_COVERAGE_PATH" \
      "$NEW_COVERAGE_PATH" \
      "$CHANGED_FILES_PATH")
  end_group

  if [ -z "$REPORT" ]; then
    echo "::notice::No coverage report to output"
    exit 0
  fi

  printf "%s\n%s\n" "$COMMENT_TAG" "$REPORT" > $COVERAGE_COMMENT_PATH

  # Output the coverage report as a multiline GitHub output parameter
  echo "Writing GitHub output parameter to \"$GITHUB_OUTPUT\""
  {
    echo "coverage_report<<END_OF_COVERAGE_REPORT"
    cat "$COVERAGE_COMMENT_PATH"
    echo "END_OF_COVERAGE_REPORT"
  } >> "$GITHUB_OUTPUT"

  if [ "$SKIP_COMMENT" = "true" ]; then
    echo "Skipping pull request comment (\$SKIP_COMMENT=true))"
    exit 0
  fi

  start_group "Comment on pull request"
  post_comment "$GITHUB_PULL_REQUEST_NUMBER" "$COVERAGE_COMMENT_PATH" "$COMMENT_TAG"
  end_group
}

main
