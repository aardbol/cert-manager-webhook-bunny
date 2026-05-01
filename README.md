[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/cert-manager-webhook-bunny)](https://artifacthub.io/packages/helm/cert-manager-webhook-bunny/cert-manager-webhook-bunny)
[![Go Report Card](https://goreportcard.com/badge/github.com/aardbol/cert-manager-webhook-bunny)](https://goreportcard.com/report/github.com/aardbol/cert-manager-webhook-bunny)
[![License](https://img.shields.io/github/license/aardbol/cert-manager-webhook-bunny)](https://github.com/aardbol/cert-manager-webhook-bunny/blob/main/LICENSE)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/aardbol/cert-manager-webhook-bunny)

cert-manager-webhook-bunny
===========================

[cert-manager](https://cert-manager.io) webhook implementation for use with [Bunny](https://bunny.net) provider for solving [ACME DNS-01 challenges](https://cert-manager.io/docs/configuration/acme/dns01/).

This fork takes a much simpler approach to the verification process to ensure compatibility with future DNS format changes at Bunny, by setting the Zone ID manually. 
But it comes with the downside that only one zone can be verified per Certificate solver configuration.

Usage
-----

For the bunny-specific configuration, you will need a Kubernetes secret containing the API key.

Ideally, you should provision this secret using an external secret manager like Bitwarden Secret Manager. If you need to create it manually for testing, use the following command:

```sh
 kubectl create secret generic bunny-api -n cert-manager --from-literal=api-key=<api-key-from-bunny-dashboard>
```

After creating the secret, configure your ``ClusterIssuer`` to have the following configuration (assuming the secret is called "bunny-api" and located in namespace "cert-manager"):

```yml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer # Or Issuer
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
          groupName: acme.aardbol.dev
          solverName: bunny
          config:
            secretRef: bunny-api
            secretNamespace: cert-manager
            zoneId: 123456
```
For more details, please refer to [https://cert-manager.io/docs/configuration/acme/dns01/webhook/](https://cert-manager.io/docs/configuration/acme/dns01/webhook/)

Now, the actual webhook can be installed directly from the GitHub Container Registry (OCI) via Helm:

```sh
helm install my-cert-manager-webhook-bunny oci://ghcr.io/aardbol/charts/cert-manager-webhook-bunny --version <CHART_VERSION> --namespace cert-manager
```

From that point, the issuer configured above should be able to solve the DNS01 challenges using ``cert-manager-webhook-bunny``.

Disclaimer
----------

I am in no way affiliated or associated with Bunny.

(c) David Hidvegi and contributors.

License
-------

[Apache 2 License](./LICENSE)