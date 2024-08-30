#!/usr/bin/env bash
# NOTE:
# Use /usr/bin/env to find shell interpreter for better portability.
# Reference: https://en.wikipedia.org/wiki/Shebang_%28Unix%29#Portability

# NOTE:
# Exit immediately if any commands (even in pipeline)
# exits with a non-zero status.
set -e
set -o pipefail

# WARNING:
# This is not reliable when using POSIX sh
# and current script file is sourced by `source` or `.`
CURRENT_SOURCE_FILE_PATH="${BASH_SOURCE[0]:-$0}"
CURRENT_SOURCE_FILE_NAME="$(basename -- "$CURRENT_SOURCE_FILE_PATH")"

# shellcheck disable=SC2016
USAGE="$CURRENT_SOURCE_FILE_NAME

This script prints the project version to STDOUT
for automatically creating tag in CI.

Usage:
  $CURRENT_SOURCE_FILE_NAME -h
  $CURRENT_SOURCE_FILE_NAME

Options:
  -h	Show this screen."

while getopts ':h' option; do
	case "$option" in
	h)
		echo "$USAGE"
		exit
		;;
	\?)
		printf "$CURRENT_SOURCE_FILE_NAME: Unknown option: -%s\n\n" "$OPTARG" >&2
		echo "$USAGE" >&2
		exit 1
		;;
	esac
done
shift $((OPTIND - 1))

# NOTE:
# GitHub actions sets CI environment variable.
# Reference: https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/store-information-in-variables#default-environment-variables
if [ -z "$CI" ]; then
	echo "WARNING: This script is meant to be run in CI environment." >&2
fi

CURRENT_SOURCE_FILE_DIR="$(dirname -- "$CURRENT_SOURCE_FILE_PATH")"

cd "$CURRENT_SOURCE_FILE_DIR"

./print-variable.make INPUT=../Makefile PROJECT_VERSION
