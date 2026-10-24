#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
: "${PGHOST:=localhost}" "${PGPORT:=5432}" "${PGUSER:=postgres}"
export PGHOST PGPORT PGUSER
command -v psql >/dev/null
command -v createdb >/dev/null
command -v dropdb >/dev/null
scratch=$(mktemp -d)
active_database=""
cleanup() {
  if [ -n "$active_database" ]; then dropdb --if-exists --force "$active_database" || true; fi
  rm -rf "$scratch"
}
trap cleanup EXIT
cat > "$scratch/init.go" <<'GO'
package main
import("log";"os";"github.com/arzumanabbasov/blacksand/db")
func main(){d,e:=db.Open(os.Getenv("BLACKSAND_PG_DSN"));if e!=nil{log.Fatal(e)};defer d.Close();if e=d.EnsureLLMUsageTable();e!=nil{log.Fatal(e)};if e=d.EnsureLLMRecordsTable();e!=nil{log.Fatal(e)}}
GO
go build -o "$scratch/init-db" "$scratch/init.go"
psql -d postgres -c 'SELECT 1'
index=0
status=0
while IFS= read -r package; do
  index=$((index + 1))
  active_database="blacksand_ci_${index}"
  createdb "$active_database"
  export BLACKSAND_PG_DSN="postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${active_database}?sslmode=disable"
  "$scratch/init-db"
  if ! BLACKSAND_ADAPTER_E2E=0 go test -p 1 "$package" -count=1; then status=1; fi
  dropdb --force "$active_database"
  active_database=""
done < <(go list ./...)
active_database="blacksand_adapter_ci"
createdb "$active_database"
export BLACKSAND_PG_DSN="postgres://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${active_database}?sslmode=disable"
"$scratch/init-db"
if ! BLACKSAND_ADAPTER_E2E=1 go test ./server -run '^TestAgentAdapterE2E$' -count=1; then status=1; fi
exit "$status"
