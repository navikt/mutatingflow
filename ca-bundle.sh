#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

ROOT=$(cd $(dirname $0)/../../; pwd)

export CA_BUNDLE=$(kubectl get configmap -n kube-system extension-apiserver-authentication -o=jsonpath='{.data.client-ca-file}' | base64 | tr -d '\n')

sed -i "s/caBundle: .*$/caBundle: ${CA_BUNDLE}/g" ./webhook.yaml
sed -i "s/url: .*$/url: https:\/\/$(hostname -f):8443\/mutate/g" ./webhook.yaml
