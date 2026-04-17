#!/usr/bin/env bash
# baseline.sh — delegates to phase 5B's k6 script, extracts latency +
# throughput, stores a versioned entry in docs/perf/baseline.json, and
# (optionally) posts a PR comment if the run diverges >10% from the
# last committed baseline on any tracked metric.
#
# Flags:
#   --dry-run       parse the k6 script but do not run it (used in CI to
#                   assert the script syntax on PRs)
#   --base URL      backend base URL (default http://localhost:8080)
#   --skip-k6       skip the run, used when k6 is unavailable
#   --update        commit a new entry to docs/perf/baseline.json
#   --compare       compare the latest result against the last committed
#                   baseline — exit 2 on >10% regression
set -euo pipefail

REPO_ROOT=$(git -C "$(dirname "$0")/../.." rev-parse --show-toplevel 2>/dev/null || echo "$(dirname "$0")/../..")
BASELINE=$REPO_ROOT/docs/perf/baseline.json
K6_SCRIPT=$REPO_ROOT/scripts/perf/k6-search.js

DRY_RUN=0
BASE=http://localhost:8080
SKIP_K6=0
UPDATE=0
COMPARE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=1; shift ;;
    --base) BASE="$2"; shift 2 ;;
    --skip-k6) SKIP_K6=1; shift ;;
    --update) UPDATE=1; shift ;;
    --compare) COMPARE=1; shift ;;
    -h|--help) grep '^#' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 1 ;;
  esac
done

# --dry-run: just assert the k6 script parses.
if [[ $DRY_RUN -eq 1 ]]; then
  if [[ ! -f "$K6_SCRIPT" ]]; then
    echo "::warning::$K6_SCRIPT not present yet (phase 5B owns it). Dry run skipped."
    exit 0
  fi
  if command -v k6 >/dev/null 2>&1; then
    # k6 has no standalone --parse, so we use --iterations=0 to force
    # it to load + compile the script without executing work.
    k6 run --quiet --iterations=0 "$K6_SCRIPT" --vus=1 2>&1 | tail -20 || {
      echo "::error::k6 failed to parse $K6_SCRIPT"
      exit 1
    }
    echo "k6 script parses OK"
  else
    # Fall back to node — the script is ESM so syntax-check via node
    # --check on a CommonJS-wrapped copy. Best effort only.
    if command -v node >/dev/null 2>&1; then
      tmp=$(mktemp --suffix=.mjs)
      # Strip k6 imports so node doesn't choke. We only want syntax.
      sed 's|from.*k6.*;||g' "$K6_SCRIPT" > "$tmp"
      node --check "$tmp" 2>&1 || {
        echo "::error::k6 script has a syntax error"
        rm -f "$tmp"
        exit 1
      }
      rm -f "$tmp"
      echo "k6 script syntax-checks via node"
    else
      echo "::warning::neither k6 nor node available — skipping dry run"
    fi
  fi
  exit 0
fi

# Actual run.
if [[ $SKIP_K6 -ne 1 ]]; then
  if [[ ! -f "$K6_SCRIPT" ]]; then
    echo "::error::$K6_SCRIPT not found (phase 5B ships it). Cannot record baseline."
    exit 1
  fi
  if ! command -v k6 >/dev/null 2>&1; then
    echo "::error::k6 not installed — install from https://k6.io/docs/getting-started/installation/"
    exit 1
  fi
  TMP=$(mktemp)
  trap 'rm -f "$TMP"' EXIT
  K6_BASE=$BASE k6 run --summary-export="$TMP" "$K6_SCRIPT"

  # Extract metrics (shape depends on phase 5B k6 script; we use the
  # standard summary-export fields).
  p50=$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(round(d["metrics"]["http_req_duration"]["p(50)"],1))' "$TMP" 2>/dev/null || echo 0)
  p95=$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(round(d["metrics"]["http_req_duration"]["p(95)"],1))' "$TMP" 2>/dev/null || echo 0)
  p99=$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(round(d["metrics"]["http_req_duration"]["p(99)"],1))' "$TMP" 2>/dev/null || echo 0)
  throughput=$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); print(round(d["metrics"]["http_reqs"]["rate"],1))' "$TMP" 2>/dev/null || echo 0)
  err=$(python3 -c 'import json,sys; d=json.load(open(sys.argv[1])); v=d["metrics"].get("http_req_failed",{}).get("rate",0); print(round(v,5))' "$TMP" 2>/dev/null || echo 0)
else
  echo "::warning::--skip-k6 set; using last recorded metrics"
  p50=$(jq -r '.[-1].metrics.p50_ms' "$BASELINE" 2>/dev/null || echo 0)
  p95=$(jq -r '.[-1].metrics.p95_ms' "$BASELINE" 2>/dev/null || echo 0)
  p99=$(jq -r '.[-1].metrics.p99_ms' "$BASELINE" 2>/dev/null || echo 0)
  throughput=$(jq -r '.[-1].metrics.throughput_rps' "$BASELINE" 2>/dev/null || echo 0)
  err=$(jq -r '.[-1].metrics.error_rate' "$BASELINE" 2>/dev/null || echo 0)
fi

commit=$(git -C "$REPO_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")
date=$(date -u +%F)
entry=$(python3 -c "
import json,sys
print(json.dumps({
  'date': '$date',
  'commit': '$commit',
  'metrics': {
    'p50_ms': float('$p50'),
    'p95_ms': float('$p95'),
    'p99_ms': float('$p99'),
    'throughput_rps': float('$throughput'),
    'error_rate': float('$err'),
  }
}, indent=2))
")

echo "Current run:"
echo "$entry"

if [[ $COMPARE -eq 1 && -f "$BASELINE" ]]; then
  last=$(python3 -c "import json; print(json.dumps(json.load(open('$BASELINE'))[-1]))" 2>/dev/null || echo '{}')
  if [[ "$last" != "{}" ]]; then
    regressed=$(python3 -c "
import json,sys
a=json.loads('''$last''')['metrics']
b=json.loads('''$entry''')['metrics']
out=[]
for k in ('p50_ms','p95_ms','p99_ms','throughput_rps'):
  old=a.get(k,0) or 0; new=b.get(k,0) or 0
  if old == 0: continue
  if k.startswith('p'):
    # higher latency = worse
    if new > old * 1.10:
      out.append(f'{k}: {old:.1f} -> {new:.1f} (+{(new-old)/old*100:.1f}%)')
  else:
    # lower throughput = worse
    if new < old * 0.90:
      out.append(f'{k}: {old:.1f} -> {new:.1f} ({(new-old)/old*100:.1f}%)')
print('\n'.join(out))
")
    if [[ -n "$regressed" ]]; then
      echo
      echo "::error::perf regression >10% against last baseline:"
      echo "$regressed"
      exit 2
    fi
    echo "No regression against last baseline."
  fi
fi

if [[ $UPDATE -eq 1 ]]; then
  if [[ ! -f "$BASELINE" ]]; then
    echo "[]" > "$BASELINE"
  fi
  python3 -c "
import json, sys
arr=json.load(open('$BASELINE'))
arr.append(json.loads('''$entry'''))
json.dump(arr, open('$BASELINE','w'), indent=2)
open('$BASELINE','a').write('\n')
"
  echo "Appended entry to $BASELINE"
fi

exit 0
