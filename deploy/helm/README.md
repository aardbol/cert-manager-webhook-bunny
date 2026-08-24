cert-manager-webhook-bunny
===========================

[cert-manager](https://cert-manager.io) webhook implementation for use with [Bunny](https://bunny.net) provider for solving
[ACME DNS-01 challenges](https://cert-manager.io/docs/configuration/acme/dns01/).

This fork takes a simpler approach by requiring the Bunny DNS Zone ID to be
set manually. This avoids depending on Bunny zone lookup behavior and should
be more robust against future Bunny DNS API or response format changes.

The trade-off is that one solver configuration targets one Bunny zone, so with
the current implementation only one zone can be handled per certificate solver configuration.

Usage
-----

For the Bunny-specific configuration, you will need to create a Kubernetes Secret containing the API key.

You can do it like this:

```sh
kubectl create secret generic bunny-api -n cert-manager --from-literal=api-key=<api-key-from-bunny-dashboard>
```

You can prepend the command with a space so that it is not saved in your shell history, depending on shell support. 
Prefer using an external secret manager where possible.

After creating the Secret, configure your ``Issuer`` or ``ClusterIssuer``.
The example below assumes the Secret is called ``bunny-api`` and located in namespace ``cert-manager``.

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
            groupName: bunny.aardbol.dev
            solverName: bunny
            config:
              secretRef: bunny-api
              secretNamespace: cert-manager
```

The Secret may contain:
- ``secretKey``: the key in the Secret that contains the Bunny API key, defaults to ``api-key``.

The webhook config must contain:
- ``secretRef``: the name of the Secret containing the Bunny API key, as created above.

The webhook config may also contain:
- ``secretNamespace``: the namespace where the Secret is located, defaults to ``cert-manager``.

The ``groupName`` must match the Helm chart value used when deploying the webhook.

For more details, please refer to https://cert-manager.io/docs/configuration/acme/dns01/webhook/

Helm installation
-----------------

Install directly from the GitHub Container Registry (OCI):

```sh
helm install cert-manager-webhook-bunny oci://ghcr.io/aardbol/charts/cert-manager-webhook-bunny --version <CHART_VERSION> -n cert-manager
```

You can also override values explicitly:

```sh
helm install cert-manager-webhook-bunny oci://ghcr.io/aardbol/charts/cert-manager-webhook-bunny \
  --version <CHART_VERSION> \
  -n cert-manager \
  --set groupName=bunny.aardbol.dev \
  --set secretAccess.namespace=cert-manager
```

From that point, the issuer configured above should be able to solve DNS01 challenges using ``cert-manager-webhook-bunny``.

Values
------

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| groupName | string | `"bunny.aardbol.dev"` | API group used by cert-manager to find this webhook solver |
| replicaCount | int | `1` | Number of webhook replicas |
| certManager.namespace | string | `"cert-manager"` | Namespace where cert-manager runs |
| certManager.serviceAccountName | string | `"cert-manager"` | Service account used by cert-manager |
| image.repository | string | `"ghcr.io/aardbol/cert-manager-webhook-bunny"` | Container image repository |
| image.tag | string | `""` | Container image tag (defaults to appVersion from Chart.yaml) |
| image.pullPolicy | string | `"IfNotPresent"` | Container image pull policy |
| nameOverride | string | `""` | Override the chart name |
| fullnameOverride | string | `""` | Override the full resource names |
| service.type | string | `"ClusterIP"` | Kubernetes service type |
| service.port | int | `443` | Kubernetes service port |
| secretAccess.namespace | string | `"cert-manager"` | Namespace where Bunny API secrets are allowed to be read from |
| secretAccess.names | list | `[]` | List of Secret names the webhook may read (empty = any Secret in namespace) |
| resources | object | `{}` | Resource requests and limits |
| nodeSelector | object | `{}` | Node selector for pod scheduling |
| tolerations | list | `[]` | Tolerations for pod scheduling |
| affinity | object | `{}` | Affinity rules for pod scheduling |

Notes
-----

The chart RBAC is scoped to the namespace configured in``secretAccess.namespace``. 
If your ``secretNamespace`` in the Issuer points to a different namespace, the webhook will not be allowed to read the Secret.

License
-------

[Apache 2 License](https://github.com/aardbol/cert-manager-webhook-bunny/blob/main/LICENSE)