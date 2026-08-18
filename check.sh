#!/usr/bin/env bash
#
# Checks your exercises. Run from the "Learning Go" directory:
#
#   ./check.sh              # check everything
#   ./check.sh 3            # check just 03_functions
#   ./check.sh 03 07        # check a few
#   ./check.sh basics       # exercises 01-09
#   ./check.sh concurrency  # exercises 10-17
#   ./check.sh practitioner # exercises 18-25
#   ./check.sh -v           # show output for passing exercises too
#
# Everything runs under -race. In the concurrency set that is not optional:
# a missing lock usually still produces the right answer, and the race
# detector is the only thing that reliably says otherwise.
#
set -uo pipefail
cd "$(dirname "$0")"

verbose=0
args=()
for a in "$@"; do
  case "$a" in
    -v)          verbose=1 ;;
    basics)      args+=(01 02 03 04 05 06 07 08 09) ;;
    concurrency) args+=(10 11 12 13 14 15 16 17) ;;
    practitioner) args+=(18 19 20 21 22 23 24 25) ;;
    *)           args+=("$a") ;;
  esac
done

# Resolve arguments like "3" or "03" or "03_functions" to directory names.
dirs=()
if [ ${#args[@]} -eq 0 ]; then
  for d in [0-9][0-9]_*/; do dirs+=("${d%/}"); done
else
  for a in "${args[@]}"; do
    match=""
    for d in [0-9][0-9]_*/; do
      d="${d%/}"
      if [ "$d" = "$a" ] || [ "${d%%_*}" = "$a" ] || [ "${d%%_*}" = "0$a" ]; then
        match="$d"; break
      fi
    done
    if [ -z "$match" ]; then
      echo "no exercise matching '$a'" >&2
      exit 2
    fi
    dirs+=("$match")
  done
fi

if [ -t 1 ]; then
  green=$'\033[32m'; red=$'\033[31m'; yellow=$'\033[33m'; dim=$'\033[2m'; off=$'\033[0m'
else
  green=""; red=""; yellow=""; dim=""; off=""
fi

failed=()
todo=()

for d in "${dirs[@]}"; do
  out=$(go test -race -timeout 180s "./$d" 2>&1)
  code=$?

  if [ $code -eq 0 ]; then
    printf '%s  PASS%s  %s\n' "$green" "$off" "$d"
    if [ $verbose -eq 1 ]; then printf '%s%s%s\n' "$dim" "$(sed 's/^/        /' <<<"$out")" "$off"; fi
    continue
  fi

  # A compile error means the code you need hasn't been written yet (or has a
  # different signature) — worth calling out differently from a failed check.
  if grep -q '\[build failed\]' <<<"$out"; then
    printf '%s  TODO%s  %s %s(does not compile yet)%s\n' "$yellow" "$off" "$d" "$dim" "$off"
    todo+=("$d")

    # The checker names every function/type it expects. Collapse the repeats.
    undef=$(grep -o 'undefined: [A-Za-z_][A-Za-z0-9_]*' <<<"$out" \
            | sed 's/undefined: //' | sort -u | tr '\n' ' ' | sed 's/ *$//')
    if [ -n "$undef" ]; then
      printf '        not written yet: %s\n' "$undef"
      if grep -q 'too many errors' <<<"$out"; then
        printf '        %s(the compiler stopped after 10 errors — there may be more)%s\n' "$dim" "$off"
      fi
    fi

    # Any other compile errors, one per line, deduped.
    others=$(grep -E '\.go:[0-9]+:[0-9]+:' <<<"$out" \
             | grep -v -e 'undefined:' -e 'too many errors' \
             | sed 's/^[[:space:]]*//' | sort -u)
    if [ -n "$others" ]; then
      sed 's/^/        /' <<<"$others"
    fi
    echo
    continue
  fi

  printf '%s  FAIL%s  %s\n' "$red" "$off" "$d"
  failed+=("$d")
  sed 's/^/        /' <<<"$out"
  echo
done

echo
total=${#dirs[@]}
remaining=("${failed[@]:-}" "${todo[@]:-}")
names=""
for n in "${remaining[@]}"; do [ -n "$n" ] && names="${names:+$names }$n"; done

if [ -z "$names" ]; then
  printf '%sall %d checked, all passing%s\n' "$green" "$total" "$off"
  exit 0
fi
bad=$(( ${#failed[@]} + ${#todo[@]} ))
printf '%s%d of %d not passing yet: %s%s\n' "$red" "$bad" "$total" "$names" "$off"
exit 1
