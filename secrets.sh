kubectl create secret -n kube-system generic rancher-api-secrets \
--from-literal=RANCHER_SERVER_URL='https://<YOUR_RANCHER_FQDN>' \
--from-literal=RANCHER_TOKEN_KEY='token-8nnsb:564jvd9dzb8754rt92ksdpnmcjrl9vx4s2jpzcjbwg5b6w4tbzsnqr'