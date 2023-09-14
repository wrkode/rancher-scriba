kubectl create secret -n kube-system generic rancher-api-secrets \
--from-literal=RANCHER_SERVER_URL='https://<YOUR_RANCHER_FQDN/v3' \
--from-literal=RANCHER_TOKEN_KEY='token-p8q56:887b578h5djp6tpttkfpflh7ztzdtx6x2dgwr96gsd5kdjmxjmttbs'