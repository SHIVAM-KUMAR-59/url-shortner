#!/usr/bin/env bash
#
# Proper load test suite for the URL shortener.
#
# Usage:
#   ./load_test.sh <api_key>
#
# Requires: hey (https://github.com/rakyll/hey), curl, jq (optional, falls
# back to grep/sed if jq isn't installed)
#
# IMPORTANT: temporarily raise your authenticated rate limit before running
# this (e.g. to 100000/min) and restart your server, or this script's own
# requests will be rate-limited and skew results.

set -uo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
API_KEY="${1:?Usage: ./load_test.sh <api_key>}"
SEED_COUNT="${SEED_COUNT:-200}"
URL_LIST_FILE="$(mktemp)"

echo "=== Seeding $SEED_COUNT short URLs ==="

for i in $(seq 1 "$SEED_COUNT"); do
	resp=$(curl -s -X POST "$BASE_URL/api/v1/shorten" \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $API_KEY" \
		-d "{\"long_url\": \"https://loadtest-seed.com/$i\"}")

	# Extract short_code without requiring jq.
	code=$(echo "$resp" | sed -n 's/.*"short_code":"\([^"]*\)".*/\1/p')

	if [[ -z "$code" ]]; then
		echo "  seed $i failed, response: $resp"
	else
		echo "$BASE_URL/$code" >> "$URL_LIST_FILE"
	fi
done

seeded=$(wc -l < "$URL_LIST_FILE" | tr -d ' ')
echo "Seeded $seeded URLs into $URL_LIST_FILE"
echo

if [[ "$seeded" -eq 0 ]]; then
	echo "No URLs seeded successfully — aborting."
	exit 1
fi

echo "=== Test 1: Multi-key redirect load (realistic cache mix) ==="
echo "hey has no native multi-URL mode, so we sample a subset of the"
echo "seeded keys (up to 20) and hit each for a few seconds — enough"
echo "keys to force real cache variety without diluting each run to"
echo "statistical noise."
echo

SAMPLE_SIZE=20
SAMPLE_FILE="$(mktemp)"
shuf -n "$SAMPLE_SIZE" "$URL_LIST_FILE" 2>/dev/null > "$SAMPLE_FILE" \
	|| head -n "$SAMPLE_SIZE" "$URL_LIST_FILE" > "$SAMPLE_FILE"
sample_count=$(wc -l < "$SAMPLE_FILE" | tr -d ' ')
PER_KEY_SECONDS=$(( 60 / sample_count > 1 ? 60 / sample_count : 2 ))

: > /tmp/loadtest_multi_results.txt
while read -r url; do
	hey -z "${PER_KEY_SECONDS}s" -c 10 -disable-redirects "$url" \
		>> /tmp/loadtest_multi_results.txt 2>&1
done < "$SAMPLE_FILE"

echo "Aggregate requests/sec across all keys:"
grep "Requests/sec" /tmp/loadtest_multi_results.txt \
	| awk '{sum += $2; count++} END {print "  total sum:", sum, " avg per key:", sum/count}'
echo "Status code distribution (all keys combined):"
grep -A2 "Status code distribution" /tmp/loadtest_multi_results.txt \
	| grep "^\s*\[" | awk '{codes[$1]+=$2} END {for (c in codes) print " ", c, codes[c]}'
echo

echo "=== Test 2: Write path load (POST /api/v1/shorten) ==="
hey -z 30s -c 20 -m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $API_KEY" \
	-d '{"long_url": "https://loadtest-write.com/x"}' \
	"$BASE_URL/api/v1/shorten"
echo

echo "=== Test 3: Combined load (reads + writes simultaneously) ==="
echo "Starting write load in background, running read load in foreground..."

hey -z 30s -c 20 -m POST \
	-H "Content-Type: application/json" \
	-H "Authorization: Bearer $API_KEY" \
	-d '{"long_url": "https://loadtest-combined.com/x"}' \
	"$BASE_URL/api/v1/shorten" > /tmp/loadtest_combined_write.txt 2>&1 &
WRITE_PID=$!

# Read load: cycle through seeded URLs for the same 30s window.
FIRST_URL=$(head -n 1 "$URL_LIST_FILE")
hey -z 30s -c 50 -disable-redirects "$FIRST_URL" > /tmp/loadtest_combined_read.txt 2>&1

wait "$WRITE_PID"

echo "--- Read results (during combined load) ---"
grep -E "Requests/sec|Slowest|Fastest|Average" /tmp/loadtest_combined_read.txt
echo
echo "--- Write results (during combined load) ---"
grep -E "Requests/sec|Slowest|Fastest|Average" /tmp/loadtest_combined_write.txt
echo

echo "Done. Seeded URLs list left at: $URL_LIST_FILE"
echo "Remember to revert your rate limit change and restart the server."