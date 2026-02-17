#!/bin/bash
set -euo pipefail

echo "Deploying to production..."
docker push registry.example.com/app:latest
curl -X POST https://deploy.example.com/trigger \
  -H "Authorization: Bearer $DEPLOY_TOKEN"
