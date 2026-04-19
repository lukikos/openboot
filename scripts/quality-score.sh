#!/usr/bin/env bash
# quality-score.sh — OpenBoot quality scorecard runner
# Usage: bash scripts/quality-score.sh
# Outputs a formatted report and writes quality/score.json

set -euo pipefail

export PATH="${PATH}:${HOME}/go/bin:/usr/local/go/bin"

# ---------------------------------------------------------------------------
# Color helpers
# ---------------------------------------------------------------------------
if [ -t 1 ] && command -v tput &>/dev/null && tput colors &>/dev/null; then
  BOLD=$(tput bold)
  RESET=$(tput sgr0)
  RED=$(tput setaf 1)
  YELLOW=$(tput setaf 3)
  GREEN=$(tput setaf 2)
  CYAN=$(tput setaf 6)
  MAGENTA=$(tput setaf 5)
  DIM=$(tput dim 2>/dev/null || true)
else
  BOLD=""
  RESET=""
  RED=""
  YELLOW=""
  GREEN=""
  CYAN=""
  MAGENTA=""
  DIM=""
fi

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
QUALITY_DIR="${ROOT_DIR}/quality"
COVERAGE_OUT="${QUALITY_DIR}/coverage.out"
SCORE_JSON="${QUALITY_DIR}/score.json"

mkdir -p "${QUALITY_DIR}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
section() { echo; echo "${BOLD}${CYAN}── $1 ${RESET}"; }

# Clamp a value between lo and hi
clamp() {
  local val=$1 lo=$2 hi=$3
  awk "BEGIN { v=$val; if(v<$lo) v=$lo; if(v>$hi) v=$hi; print v }"
}

# Linear interpolation between score bands.
# score_for <value> <perfect> <good> <acceptable> <poor> <higher_is_better>
# higher_is_better: 1 = higher value is better (coverage), 0 = lower is better (errors)
score_for() {
  local val=$1 perfect=$2 good=$3 acceptable=$4 poor=$5 hib=$6
  awk -v v="$val" -v p="$perfect" -v g="$good" -v a="$acceptable" -v po="$poor" -v hib="$hib" '
  function lerp(lo, hi, lo_score, hi_score, x) {
    if (hi == lo) return lo_score
    return lo_score + (x - lo) / (hi - lo) * (hi_score - lo_score)
  }
  BEGIN {
    if (hib == 1) {
      # higher value = better score
      if      (v >= p)  { score = 10 }
      else if (v >= g)  { score = lerp(g, p, 8, 10, v) }
      else if (v >= a)  { score = lerp(a, g, 6, 8,  v) }
      else if (v >= po) { score = lerp(po, a, 4, 6, v) }
      else              { score = lerp(0, po, 0, 4, v) }
      if (score < 0) score = 0
    } else {
      # lower value = better score
      if      (v <= p)  { score = 10 }
      else if (v <= g)  { score = lerp(p, g, 10, 8, v) }
      else if (v <= a)  { score = lerp(g, a, 8, 6,  v) }
      else if (v <= po) { score = lerp(a, po, 6, 4, v) }
      else              { score = lerp(po, po*3, 4, 0, v); if(score < 0) score = 0 }
    }
    printf "%.2f", score
  }'
}

score_color() {
  local s=$1
  awk -v s="$s" 'BEGIN {
    if      (s >= 9) { exit 0 }
    else if (s >= 7) { exit 1 }
    else if (s >= 5) { exit 2 }
    else             { exit 3 }
  }' && echo "${GREEN}" || {
    local rc=$?
    case $rc in
      1) echo "${YELLOW}" ;;
      2) echo "${YELLOW}" ;;
      *) echo "${RED}" ;;
    esac
  }
}

score_bar() {
  local score=$1  # 0-10
  local filled
  filled=$(awk "BEGIN { printf \"%d\", $score }")
  local bar=""
  for i in $(seq 1 10); do
    if [ "$i" -le "$filled" ]; then
      bar="${bar}█"
    else
      bar="${bar}░"
    fi
  done
  echo "$bar"
}

# ---------------------------------------------------------------------------
# 1. Test coverage
# ---------------------------------------------------------------------------
section "Test Coverage"
COVERAGE_PCT="N/A"
COVERAGE_SCORE="0"

if go test ./... -coverprofile="${COVERAGE_OUT}" -covermode=atomic \
    -timeout 5m 2>/dev/null; then
  raw=$(go tool cover -func="${COVERAGE_OUT}" 2>/dev/null | \
        grep -E "^total:" | awk '{print $3}' | tr -d '%')
  if [ -n "$raw" ]; then
    COVERAGE_PCT="${raw}%"
    COVERAGE_SCORE=$(score_for "$raw" 90 80 70 60 1)
    echo "  Coverage : ${BOLD}${COVERAGE_PCT}${RESET}"
  fi
else
  echo "  ${YELLOW}Tests failed or timed out — coverage unavailable${RESET}"
fi

# ---------------------------------------------------------------------------
# 2. Lint errors
# ---------------------------------------------------------------------------
section "Lint (golangci-lint)"
LINT_COUNT="N/A"
LINT_SCORE="5"   # neutral default when tool is absent

if command -v golangci-lint &>/dev/null; then
  lint_output=$(golangci-lint run ./... 2>/dev/null)
  lint_exit=$?
  lint_tool_ok=true
  if [ $lint_exit -eq 0 ]; then
    raw=0
  elif [ $lint_exit -eq 1 ]; then
    raw=$(echo "$lint_output" | grep -c "." || echo 0)
  else
    echo "  ${YELLOW}golangci-lint tool error (exit ${lint_exit}) — skipping lint score${RESET}"
    lint_tool_ok=false
  fi
  if $lint_tool_ok; then
    LINT_COUNT="${raw}"
    LINT_SCORE=$(score_for "$raw" 0 3 8 15 0)
    echo "  Issues   : ${BOLD}${LINT_COUNT}${RESET}"
  fi
else
  echo "  ${DIM}golangci-lint not installed — skipping (score neutral 5/10)${RESET}"
  LINT_COUNT="N/A"
fi

# ---------------------------------------------------------------------------
# 3. Security (gosec)
# ---------------------------------------------------------------------------
section "Security (gosec)"
SEC_COUNT="N/A"
SEC_SCORE="5"   # neutral default

if command -v gosec &>/dev/null; then
  raw=$(gosec -quiet ./... 2>&1 | grep -c "Issue" || true)
  SEC_COUNT="${raw}"
  SEC_SCORE=$(score_for "$raw" 0 1 3 6 0)
  echo "  Issues   : ${BOLD}${SEC_COUNT}${RESET}"
else
  echo "  ${DIM}gosec not installed — skipping (score neutral 5/10)${RESET}"
  SEC_COUNT="N/A"
fi

# ---------------------------------------------------------------------------
# 4. Cyclomatic complexity
# ---------------------------------------------------------------------------
section "Complexity (gocyclo)"
COMPLEXITY_AVG="N/A"
COMPLEXITY_SCORE="5"

if command -v gocyclo &>/dev/null; then
  raw=$(gocyclo -over 1 "${ROOT_DIR}/internal" "${ROOT_DIR}/cmd" 2>/dev/null \
        | grep -v "vendor\|testutil\|_test\.go" \
        | awk '{sum+=$1; count++} END { if(count>0) printf "%.2f", sum/count; else print "0" }' || true)
  if [ -n "$raw" ] && [ "$raw" != "0" ]; then
    COMPLEXITY_AVG="${raw}"
    COMPLEXITY_SCORE=$(score_for "$raw" 5 8 12 18 0)
    echo "  Avg complexity: ${BOLD}${COMPLEXITY_AVG}${RESET}"
  else
    COMPLEXITY_AVG="0"
    COMPLEXITY_SCORE="10"
    echo "  Avg complexity: ${BOLD}0${RESET} (no functions above threshold)"
  fi
else
  echo "  ${DIM}gocyclo not installed — skipping (score neutral 5/10)${RESET}"
fi

# ---------------------------------------------------------------------------
# 5. File size (files > 800 lines)
# ---------------------------------------------------------------------------
section "File Size (files > 800 lines)"
LARGE_FILES=0
LARGE_SCORE="0"

LARGE_FILES=$(find "${ROOT_DIR}" -name "*.go" \
    ! -path "*/vendor/*" \
    ! -path "*/testutil/*" \
    ! -path "*/.claude/*" \
    ! -name "*_test.go" \
    -exec wc -l {} + 2>/dev/null \
  | awk '$1 > 800 && $2 != "total" { count++ } END { print count+0 }')

LARGE_SCORE=$(score_for "$LARGE_FILES" 0 1 3 6 0)
echo "  Large files: ${BOLD}${LARGE_FILES}${RESET}"

# ---------------------------------------------------------------------------
# 6. Weighted total
# ---------------------------------------------------------------------------
TOTAL_SCORE=$(awk \
  -v cov="${COVERAGE_SCORE}" \
  -v lint="${LINT_SCORE}" \
  -v sec="${SEC_SCORE}" \
  -v cmp="${COMPLEXITY_SCORE}" \
  -v fsz="${LARGE_SCORE}" \
  'BEGIN {
    total = cov*25 + lint*20 + sec*25 + cmp*15 + fsz*15
    printf "%.1f", total / 100
  }')

# ---------------------------------------------------------------------------
# 7. Report
# ---------------------------------------------------------------------------
echo
echo "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo "${BOLD}${MAGENTA}  OpenBoot Quality Scorecard${RESET}"
echo "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
printf "  %-22s  %s  %s  %s\n" "Dimension" "Score" "Bar" "Weight"
echo "  ──────────────────────────────────────────────"

print_row() {
  local name=$1 score=$2 weight=$3 raw_val=$4
  local bar
  bar=$(score_bar "$score")
  local col
  col=$(score_color "$score")
  printf "  %-22s  ${col}%4s${RESET}  %s  %3s%%   %s\n" \
    "$name" "${score}/10" "${bar}" "$weight" "(raw: ${raw_val})"
}

print_row "test_coverage"  "${COVERAGE_SCORE}"   25 "${COVERAGE_PCT}"
print_row "lint_errors"    "${LINT_SCORE}"        20 "${LINT_COUNT}"
print_row "security"       "${SEC_SCORE}"         25 "${SEC_COUNT}"
print_row "complexity"     "${COMPLEXITY_SCORE}"  15 "${COMPLEXITY_AVG}"
print_row "file_size"      "${LARGE_SCORE}"       15 "${LARGE_FILES} files"

echo "  ──────────────────────────────────────────────"

TOTAL_COLOR=$(score_color "$TOTAL_SCORE")
printf "  %-22s  ${TOTAL_COLOR}${BOLD}%s${RESET}\n" "TOTAL SCORE" "${TOTAL_SCORE} / 10"
echo "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo

# ---------------------------------------------------------------------------
# 8. Write score.json
# ---------------------------------------------------------------------------
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo "unknown")

cat > "${SCORE_JSON}" <<JSON
{
  "timestamp": "${TIMESTAMP}",
  "git_sha": "${GIT_SHA}",
  "dimensions": {
    "test_coverage": {
      "raw": "${COVERAGE_PCT}",
      "score": ${COVERAGE_SCORE},
      "weight": 25
    },
    "lint_errors": {
      "raw": "${LINT_COUNT}",
      "score": ${LINT_SCORE},
      "weight": 20
    },
    "security": {
      "raw": "${SEC_COUNT}",
      "score": ${SEC_SCORE},
      "weight": 25
    },
    "complexity": {
      "raw": "${COMPLEXITY_AVG}",
      "score": ${COMPLEXITY_SCORE},
      "weight": 15
    },
    "file_size": {
      "raw": "${LARGE_FILES}",
      "score": ${LARGE_SCORE},
      "weight": 15
    }
  },
  "total_score": ${TOTAL_SCORE}
}
JSON

echo "  Score saved to ${SCORE_JSON}"
echo
