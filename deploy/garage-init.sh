#!/bin/sh
# One-shot Garage setup, run by the garage-init service. Safe to re-run.
#
# - gives the single node a role in the cluster layout (first start only)
# - creates the photo bucket and enables website access on it; that is how
#   caddy serves /img/* read-only (Garage has no anonymous-read bucket policy)
# - imports the api's access key with read/write on that bucket only
set -eu

: "${GARAGE_ADMIN_TOKEN:?}" "${S3_BUCKET:?}" "${S3_ACCESS_KEY:?}" "${S3_SECRET_KEY:?}"

ADMIN=http://garage:3903
AUTH="Authorization: Bearer $GARAGE_ADMIN_TOKEN"

get() { curl -fsS -H "$AUTH" "$ADMIN/v2/$1"; }
post() { curl -fsS -H "$AUTH" -H 'Content-Type: application/json' --data "$2" "$ADMIN/v2/$1"; }

# field NAME: first value of NAME (string or number) in the JSON on stdin.
field() { tr -d '\n' | grep -o "\"$1\": *\"\{0,1\}[^\",}]*" | head -n1 | sed 's/.*: *"\{0,1\}//'; }

until status=$(get GetClusterStatus 2>/dev/null); do
  echo "waiting for garage..."
  sleep 1
done

if [ "$(echo "$status" | field layoutVersion)" = "0" ]; then
  node=$(echo "$status" | field id)
  post UpdateClusterLayout "{\"roles\":[{\"id\":\"$node\",\"zone\":\"local\",\"capacity\":10000000000,\"tags\":[]}]}" >/dev/null
  post ApplyClusterLayout '{"version":1}' >/dev/null
  echo "garage-init: layout applied for node $node"
fi

until curl -fsS "$ADMIN/health" >/dev/null 2>&1; do
  echo "waiting for garage to become healthy..."
  sleep 1
done

if ! bucket=$(get "GetBucketInfo?globalAlias=$S3_BUCKET" 2>/dev/null); then
  bucket=$(post CreateBucket "{\"globalAlias\":\"$S3_BUCKET\"}")
fi
bucket_id=$(echo "$bucket" | field id)
post "UpdateBucket?id=$bucket_id" '{"websiteAccess":{"enabled":true,"indexDocument":"index.html"}}' >/dev/null

if ! get "GetKeyInfo?id=$S3_ACCESS_KEY" >/dev/null 2>&1; then
  post ImportKey "{\"accessKeyId\":\"$S3_ACCESS_KEY\",\"secretAccessKey\":\"$S3_SECRET_KEY\",\"name\":\"cartograph-api\"}" >/dev/null
fi
post AllowBucketKey "{\"bucketId\":\"$bucket_id\",\"accessKeyId\":\"$S3_ACCESS_KEY\",\"permissions\":{\"read\":true,\"write\":true,\"owner\":false}}" >/dev/null

echo "garage-init: bucket '$S3_BUCKET' ready"
