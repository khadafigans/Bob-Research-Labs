#!/usr/bin/env bash
# =============================================================================
#
#    ARTEX  .  bcrypt   settings 
# auth.password_hash  .      pgcrypto  DB  bcrypt  
#    .   (golang.org/x/crypto/bcrypt)  .
#
#                      > --dsn/$ARTEX_PG_DSN > config.json  database.*
#   docker        -- `docker compose exec`( `docker exec`) postgres
#
#   ./reset-password.sh                          # , config.json/      
#   ./reset-password.sh -p 'NewPass!'            # ,    
#   ./reset-password.sh --dsn postgres://u:p@h:5432/artex
#   ./reset-password.sh -H 127.0.0.1 -P 5433 -U autopentest -W pass -d artex
#   ./reset-password.sh -m docker                # docker (.env  POSTGRES_*  )
#   ./reset-password.sh -m docker -c pg --exec docker
#
# :     + psql \getenv  ( argv   ), :'var' 
#  (SQL  ).   PGPASSWORD    argv   .
# =============================================================================
set -euo pipefail

PASS_KEY="auth.password_hash"
BCRYPT_COST=10

MODE=""            # local | docker(  )
DSN=""
HOST="" PORT="" USER="" DBPASS="" DBNAME="" SSLMODE=""
CONFIG=""
CONTAINER=""       # docker  postgres /( postgres)
EXEC_KIND=""       # compose | docker(docker   exec  .  )
NEWPASS=""
ASSUME_YES=0

die() { echo ": $*" >&2; exit 1; }
info() { echo "· $*" >&2; }

usage() { sed -n '2,25p' "$0" | sed 's/^# \{0,1\}//'; exit 0; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--mode)        MODE="${2:-}"; shift 2 ;;
    --dsn)            DSN="${2:-}"; shift 2 ;;
    -H|--host)        HOST="${2:-}"; shift 2 ;;
    -P|--port)        PORT="${2:-}"; shift 2 ;;
    -U|--user)        USER="${2:-}"; shift 2 ;;
    -W|--db-password) DBPASS="${2:-}"; shift 2 ;;
    -d|--dbname)      DBNAME="${2:-}"; shift 2 ;;
    --sslmode)        SSLMODE="${2:-}"; shift 2 ;;
    --config)         CONFIG="${2:-}"; shift 2 ;;
    -c|--container)   CONTAINER="${2:-}"; shift 2 ;;
    --exec)           EXEC_KIND="${2:-}"; shift 2 ;;
    -p|--new-password) NEWPASS="${2:-}"; shift 2 ;;
    -y|--yes)         ASSUME_YES=1; shift ;;
    -h|--help)        usage ;;
  esac
done

# ---- config.json  database.* (local      ) -----
# python3    (). python3   grep  (config.json    ).
read_config_json() {
  local path="$1"
  [[ -f "$path" ]] || return 1
  if command -v python3 >/dev/null 2>&1; then
    python3 - "$path" <<'PY'
import json, sys
try:
    d = json.load(open(sys.argv[1])).get("database", {})
except Exception:
    sys.exit(1)
if d.get("dsn"):
    print("DSN\t" + d["dsn"]); sys.exit(0)
for k in ("host","port","user","password","dbname","sslmode"):
    if d.get(k) is not None:
        print(k.upper() + "\t" + str(d[k]))
PY
  else
    local k
    for k in host port user password dbname sslmode; do
      local v
      v=$(grep -oE "\"$k\"[[:space:]]*:[[:space:]]*(\"[^\"]*\"|[0-9]+)" "$path" 2>/dev/null \
            | head -1 | sed -E "s/.*:[[:space:]]*//; s/^\"//; s/\"$//") || true
      [[ -n "$v" ]] && echo -e "${k^^}\t$v"
    done
  fi
}

apply_config_fields() {
  local line key val
  while IFS=$'\t' read -r key val; do
    [[ -z "$key" ]] && continue
    case "$key" in
      DSN)      [[ -z "$DSN" ]] && DSN="$val" ;;
      HOST)     [[ -z "$HOST" ]] && HOST="$val" ;;
      PORT)     [[ -z "$PORT" ]] && PORT="$val" ;;
      USER)     [[ -z "$USER" ]] && USER="$val" ;;
      PASSWORD) [[ -z "$DBPASS" ]] && DBPASS="$val" ;;
      DBNAME)   [[ -z "$DBNAME" ]] && DBNAME="$val" ;;
      SSLMODE)  [[ -z "$SSLMODE" ]] && SSLMODE="$val" ;;
    esac
  done
}

if [[ -z "$MODE" ]]; then
  if [[ -n "$DSN$HOST$USER$DBNAME" || -n "${ARTEX_PG_DSN:-}" || -f "${CONFIG:-config.json}" ]]; then
    MODE="local"
  elif command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yml ]]; then
    MODE="docker"
  else
    MODE="local"
  fi
fi
info " : $MODE"

if [[ -z "$NEWPASS" ]]; then
  read -r -s -p "  (  ARTEX  ):" NEWPASS; echo >&2
  [[ -n "$NEWPASS" ]] || die "    "
  read -r -s -p "   :" NEWPASS2; echo >&2
  [[ "$NEWPASS" == "$NEWPASS2" ]] || die "     "
fi
[[ -n "$NEWPASS" ]] || die "    "

#    psql  (\getenv   argv/ps   )
export ARTEX_RESET_NEWPASS="$NEWPASS"

# DB  bcrypt   upsert .  :'newpw'   . CREATE EXTENSION  ,
SQL=$(cat <<SQL
\\set ON_ERROR_STOP on
\\getenv newpw ARTEX_RESET_NEWPASS
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO settings(key, value)
VALUES ('$PASS_KEY', crypt(:'newpw', gen_salt('bf', $BCRYPT_COST)))
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now();
SQL
)

if [[ "$MODE" == "local" ]]; then
  #   :  > --dsn/$ARTEX_PG_DSN > config.json
  if [[ -z "$DSN" && -z "$HOST$USER$DBNAME" ]]; then
    [[ -n "${ARTEX_PG_DSN:-}" ]] && DSN="$ARTEX_PG_DSN"
  fi
  if [[ -z "$DSN" && -z "$HOST$USER$DBNAME" ]]; then
    cfg="${CONFIG:-config.json}"
    if [[ -f "$cfg" ]]; then
      info "$cfg    "
      apply_config_fields < <(read_config_json "$cfg")
    fi
  fi

  command -v psql >/dev/null 2>&1 || die "  psql    (postgresql-client   -m docker  )"

  declare -a PSQL_ARGS=()
  if [[ -n "$DSN" ]]; then
    PSQL_ARGS=("$DSN")
    target="$DSN"
  else
    [[ -n "$USER"   ]] || die " (-U)   config.json/DSN  "
    [[ -n "$DBNAME" ]] || die " (-d)   config.json/DSN  "
    HOST="${HOST:-127.0.0.1}"; PORT="${PORT:-5432}"; SSLMODE="${SSLMODE:-disable}"
    PSQL_ARGS=(-h "$HOST" -p "$PORT" -U "$USER" -d "$DBNAME")
    [[ -n "$SSLMODE" ]] && export PGSSLMODE="$SSLMODE"
    [[ -n "$DBPASS" ]] && export PGPASSWORD="$DBPASS"
    target="$USER@$HOST:$PORT/$DBNAME"
  fi

  info " : $target"
  if [[ "$ASSUME_YES" -ne 1 ]]; then
    read -r -p "  ARTEX  ? [y/N]" ans
    [[ "$ans" == "y" || "$ans" == "Y" ]] || die ""
  fi

  if ! printf '%s\n' "$SQL" | psql "${PSQL_ARGS[@]}" -v ON_ERROR_STOP=1 -q >/dev/null; then
    die " . pgcrypto  ·          CREATE EXTENSION pgcrypto   ."
  fi

else
  # ---- docker ----
  command -v docker >/dev/null 2>&1 || die "docker    "
  CONTAINER="${CONTAINER:-postgres}"

  # exec  : docker compose exec() ,  docker exec()
  if [[ -z "$EXEC_KIND" ]]; then
    if docker compose version >/dev/null 2>&1 && [[ -f docker-compose.yml ]]; then
      EXEC_KIND="compose"
    else
      EXEC_KIND="docker"
    fi
  fi

  #   psql  :  ,  .env  POSTGRES_*,  compose (artex)
  if [[ -f .env ]]; then
    # shellcheck disable=SC1091
    set -a; . ./.env; set +a
  fi
  DUSER="${USER:-${POSTGRES_USER:-artex}}"
  DNAME="${DBNAME:-${POSTGRES_DB:-artex}}"
  [[ -n "$DBPASS" ]] && export PGPASSWORD="$DBPASS"
  [[ -z "${PGPASSWORD:-}" && -n "${POSTGRES_PASSWORD:-}" ]] && export PGPASSWORD="$POSTGRES_PASSWORD"

  info ":  $CONTAINER  psql -U $DUSER -d $DNAME(exec=$EXEC_KIND)"
  if [[ "$ASSUME_YES" -ne 1 ]]; then
    read -r -p "   ARTEX  ? [y/N]" ans
    [[ "$ans" == "y" || "$ans" == "Y" ]] || die ""
  fi

  declare -a EXEC_CMD
  if [[ "$EXEC_KIND" == "compose" ]]; then
    EXEC_CMD=(docker compose exec -T -e ARTEX_RESET_NEWPASS -e PGPASSWORD "$CONTAINER"
              psql -U "$DUSER" -d "$DNAME" -v ON_ERROR_STOP=1 -q)
  else
    EXEC_CMD=(docker exec -i -e ARTEX_RESET_NEWPASS -e PGPASSWORD "$CONTAINER"
              psql -U "$DUSER" -d "$DNAME" -v ON_ERROR_STOP=1 -q)
  fi

  if ! printf '%s\n' "$SQL" | "${EXEC_CMD[@]}" >/dev/null; then
    die " . (-c),  (.env  POSTGRES_*),  pgcrypto  ."
  fi
fi

unset ARTEX_RESET_NEWPASS
echo "✓ ARTEX   .   ARTEX +   (   )."
