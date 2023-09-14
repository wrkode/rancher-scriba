kubectl create secret -n kube-system generic rancher-api-secrets \
--from-literal=RANCHER_SERVER_URL='https://<YOUR_RANCHER_FQDN/v3' \
--from-literal=RANCHER_TOKEN_KEY='token-h7jqc:r496mwpncwjksbgdvdlmmsfhd4g56r9jtqgjqgrnbxjxzhwc9q7nk4'