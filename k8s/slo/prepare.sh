#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
context=${KUBE_CONTEXT:-sre}

umask 077
temp_dir=$(mktemp -d "${TMPDIR:-/tmp}/wishlist-prober.XXXXXX")
trap 'rm -rf -- "$temp_dir"' EXIT
trap 'exit 1' HUP INT TERM

kubectl --context="$context" -n wishlist rollout status statefulset/postgres --timeout=180s
kubectl --context="$context" -n wishlist exec -i statefulset/postgres -c postgres -- \
    sh -c 'exec psql -X -qAt -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
    < "$script_dir/seed.sql" > "$temp_dir/probe.env"

probe_url=$(sed -n 's/^PROBE_URL=//p' "$temp_dir/probe.env")
wishlist_id=$(sed -n 's/^PROBE_EXPECTED_WISHLIST_ID=//p' "$temp_dir/probe.env")
public_token=${probe_url##*/}
for value in "$wishlist_id" "$public_token"; do
    if ! printf '%s\n' "$value" | grep -Eq '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then
        echo 'Не удалось получить ID и публичный токен контрольного вишлиста' >&2
        exit 1
    fi
done

sed -e "s/CHANGE_ME_PUBLIC_TOKEN/$public_token/g" \
    -e "s/CHANGE_ME_WISHLIST_ID/$wishlist_id/g" \
    "$script_dir/prober.example.yaml" > "$temp_dir/prober.local.yaml"
mv -- "$temp_dir/prober.local.yaml" "$script_dir/prober.local.yaml"

kubectl --context="$context" apply -f "$script_dir/prober.local.yaml"
kubectl --context="$context" -n wishlist rollout restart deployment/wishlist-prober
kubectl --context="$context" -n wishlist rollout status deployment/wishlist-prober --timeout=180s
echo 'Контрольный вишлист и конфигурация пробера готовы'
