#!/bin/bash
#
# Usage:
#
# ./generate-kubernetes-resources.yml $DOMAIN $EMAIL

# 0. Check for variables
DOMAIN=$1
EMAIL=$2
if [[ -z $DOMAIN ]]; then
  echo "No 'DOMAIN' specified as the first argument"
  exit 1
fi
if [[ -z $EMAIL ]]; then
  echo "No 'EMAIL' specified as the second argument"
  exit 1
fi
# 1. Generate Private Key
openssl genrsa -out private-key.pem 2048 >/dev/null 2>&1
# Requires newlines to be encoded
PRIVATE_KEY_BASE64=$(cat ./private-key.pem | base64 -w 0)
# 2. Generate Random Port
RANDOM_INT=$(( $RANDOM % 2767 ))
NODE_PORT=$((30000 + RANDOM_INT))
# 3. Copy file
cp ./kubernetes-resources.yml-part-1.tmpl ./kubernetes-resources-part-1.yml
cp ./kubernetes-resources.yml-part-2.tmpl ./kubernetes-resources-part-2.yml
# 4. Execute replacements
sed -i -e "s/\*NODE_PORT\*/$NODE_PORT/g" kubernetes-resources-part-1.yml
sed -i -e "s/\*PRIVATE_KEY_BASE64\*/$PRIVATE_KEY_BASE64/g" kubernetes-resources-part-1.yml
sed -i -e "s/\*DOMAIN\*/$DOMAIN/g" kubernetes-resources-part-2.yml
sed -i -e "s/\*EMAIL\*/$EMAIL/g" kubernetes-resources-part-2.yml
