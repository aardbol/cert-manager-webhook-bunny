cert-manager-webhook-bunny
===========================

[cert-manager](https://cert-manager.io) webhook implementation for use with [Bunny](https://bunny.net) provider for solving
[ACME DNS-01 challenges](https://cert-manager.io/docs/configuration/acme/dns01/).

This fork takes a simpler approach by requiring the Bunny DNS Zone ID to be
set manually. This avoids depending on Bunny zone lookup behavior and should
be more robust against future Bunny DNS API or response format changes.

The trade-off is that one solver configuration targets one Bunny zone, so with
the current implementation only one zone can be handled per certificate
solver configuration.

Usage
-----

For the Bunny-specific configuration, you will need to create a Kubernetes
Secret containing the API key.

You can do it like this:

```sh
kubectl create secret generic bunny-secret -n cert-manager --from-literal=api-key=<api-key-from-bunny-dashboard>
```

You can prepend the command with a space so that it is not saved in your shell
history, depending on shell support. Prefer using an external secret manager
where possible.

After creating the Secret, configure your ``Issuer`` or ``ClusterIssuer``.
The example below assumes the Secret is called ``bunny-api`` and located
in namespace ``cert-manager``.

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
            groupName: acme.aardbol.dev
            solverName: bunny
            config:
              secretRef: bunny-api
              secretNamespace: cert-manager
              zoneId: 123456
```

The Secret must contain:
- ``api-key``

The webhook config must contain:
- ``secretRef``
- ``zoneId``

The webhook config may also contain:
- ``secretNamespace``

The ``groupName`` must match the Helm chart value used when deploying the
webhook.

For more details, please refer to https://cert-manager.io/docs/configuration/acme/dns01/webhook/

Helm installation
-----------------

Install from the Helm repository:

```sh
helm repo add cert-manager-webhook-bunny https://aardbol.github.io/cert-manager-webhook-bunny/charts/
helm repo update
helm install cert-manager-webhook-bunny cert-manager-webhook-bunny/cert-manager-webhook-bunny -n cert-manager
```

You can also override values explicitly:

```sh
helm install cert-manager-webhook-bunny cert-manager-webhook-bunny/cert-manager-webhook-bunny \
  -n cert-manager \
  --set groupName=acme.aardbol.dev \
  --set secretAccess.namespace=cert-manager
```

From that point, the issuer configured above should be able to solve DNS01
challenges using ``cert-manager-webhook-bunny``.

Notes
-----

The chart RBAC is scoped to the namespace configured in
``secretAccess.namespace``. If your ``secretNamespace`` in the Issuer points to
a different namespace, the webhook will not be allowed to read the Secret.

License
-------

[Apache 2 License](./LICENSE)