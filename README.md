[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/cert-manager-webhook-bunny)](https://artifacthub.io/packages/helm/cert-manager-webhook-bunny/cert-manager-webhook-bunny)
[![Go Report Card](https://goreportcard.com/badge/github.com/aardbol/cert-manager-webhook-bunny)](https://goreportcard.com/report/github.com/aardbol/cert-manager-webhook-bunny)
[![License](https://img.shields.io/github/license/aardbol/cert-manager-webhook-bunny)](https://github.com/aardbol/cert-manager-webhook-bunny/blob/main/LICENSE)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/aardbol/cert-manager-webhook-bunny)

cert-manager-webhook-bunny
===========================

[cert-manager](https://cert-manager.io) webhook implementation for use with [Bunny](https://bunny.net) (bunny.net) provider for solving [ACME DNS-01 challenges](https://cert-manager.io/docs/configuration/acme/dns01/).

This fork takes a much simpler approach to the verification process to ensure compatibility with future DNS
format changes at Bunny, be setting the Zone ID manually. But it comes with the downside that only one zone can be verified
per Certificate.

Usage
-----

For the bunny-specific configuration, you will need to create a Kubernetes secret, containing the API key.

You can do it like following, just place the correct values in the command:

```sh
kubectl create secret generic bunny-secret -n cert-manager --from-literal=api-key=<api-key-from-bunny-dashboard>
```
You can prepend the command with a space so that it's not saved into your terminal's history file, although this
depends on the support in your terminal's shell. Ideally, you would use an external secret manager instead.

After creating the secret, configure the ``Issuer``/``ClusterIssuer`` of
yours to have the following configuration (as assumed, secret is
called "bunny-api" and located in namespace "cert-manager"):

```yml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer # or Issuer
metadata:
  name: letsencrypt-prod-dns
spec:
  acme:
    email: your@email.pm
    privateKeySecretRef:
      name: letsencrypt-prod
    server: https://acme-v02.api.letsencrypt.org/directory
    solvers:
    - dns01:
        webhook:
            groupName: com.bunny.webhook
            solverName: bunny
            config:
                secretRef: bunny-api
                secretNamespace: cert-manager
```
For more details, please refer to https://cert-manager.io/docs/configuration/acme/dns01/#configuring-dns01-challenge-provider

Now, the actual webhook can be installed via Helm chart:
```
helm repo add cert-manager-webhook-bunny https://davidhidvegi.github.io/cert-manager-webhook-bunny/charts/

helm install my-cert-manager-webhook-bunny cert-manager-webhook-bunny/cert-manager-webhook-bunny --namespace cert-manager
```
From that point, the issuer configured above should be able to solve the DNS01 challenges using ``cert-manager-webhook-bunny``.

Disclaimer
----------

I am in no way affiliated or associated with Bunny.

License
-------

[Apache 2 License](./LICENSE)



