#!/bin/sh
# One-shot MinIO setup, run by the minio-init service. Safe to re-run.
#
# - creates the photo bucket
# - allows anonymous GET on <bucket>/photos/* only (served via caddy /img/*)
# - creates the api's access key with read/write on that bucket only
set -eu

: "${MINIO_ROOT_USER:?}" "${MINIO_ROOT_PASSWORD:?}" "${S3_BUCKET:?}"
: "${S3_ACCESS_KEY:?}" "${S3_SECRET_KEY:?}"

until mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null; do
  echo "waiting for minio..."
  sleep 1
done

mc mb --ignore-existing "local/$S3_BUCKET"
mc anonymous set download "local/$S3_BUCKET/photos"

if [ "$S3_ACCESS_KEY" = "$MINIO_ROOT_USER" ]; then
  echo "S3_ACCESS_KEY is the MinIO root user; not creating a separate api user"
else
  policy=/tmp/cartograph-rw.json
  cat > "$policy" <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": ["arn:aws:s3:::$S3_BUCKET"]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": ["arn:aws:s3:::$S3_BUCKET/*"]
    }
  ]
}
EOF
  mc admin policy create local cartograph-rw "$policy"
  mc admin user add local "$S3_ACCESS_KEY" "$S3_SECRET_KEY"
  # attach fails if the policy is already attached; that is fine on re-runs
  mc admin policy attach local cartograph-rw --user "$S3_ACCESS_KEY" 2>/dev/null || true
fi

echo "minio-init: bucket '$S3_BUCKET' ready"
