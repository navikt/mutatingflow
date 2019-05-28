Mutatingflow
============

> A mutating webhook for adding Nais-specific dependencies to Notebooks created by Kubeflow

Foreløpig er hele prosessen med utvikling, release, og deployment manuelt. Nedenfor finner man en semi-generell, og litt NAIS spesifikk deployment-pipeline. For nå så bruker vi [nais-yaml](https://github.com/navikt/nais-yaml/)-repoet for deployment av nye versjoner.


## Development and creating new release

1. Code code code
2. `make codegen-crd`
   * Kan hoppes over hvis man ikke har oppdatert notebook apiet
3. `make release docker`
4. Se *Deploy Mutatingflow*


## Deploy Mutatingflow

Hvis det ikke er første gang, hopp rett til punkt 4.

1. `kubectl apply -f deployment/secret.yaml`
2. `kubectl apply -f deployment/service.yaml`
3. `kubectl apply -f webhook.yaml`
4. Oppdatert `spec.template.spec.containers[0].image` i filen `deployment/deployment.yaml`
5. `kubectl apply -f deployment/deployment.yaml`

PS: Follow [self-signed-certificate-with-custom-ca.md](https://gist.github.com/fntlnz/cf14feb5a46b2eda428e000157447309) to create caBundle, cert, and key.


## Deploy Kubeflow on NAIS

1. Følg oppsett fra Kubeflow dokumentasjonen
  ```
  kfctl init <app> --namespace <namespace>
  kfctl generate all -V
  kfctl apply all -V
  ```
2. Sett opp basic-auth secret for ambassador
  ```
  apiVersion: v1
  kind: Secret
  metadata:
    name: ambassador-ba
  data:
    auth: <base64>
  ```
3. Sett opp ingress for ambassador.yaml (med basic-auth)
  ```
  apiVersion: extensions/v1beta1
  kind: Ingress
  metadata:
    annotations:
      ingress.kubernetes.io/auth-secret: ambassador-ba
      ingress.kubernetes.io/auth-type: basic
    name: ambassador-ingress
  spec:
    rules:
    - host: <app>-kubeflow.nais.adeo.no
      http:
        paths:
        - backend:
            serviceName: ambassador
            servicePort: 80
          path: /
  ```
3. Installer ca-bundle
   * Kopier fra et namespace som allerede har ca-bundle configmap
4. Sett opp Mutatingflow
   * Se *Deploy Mutatingflow* over


## Support

Besøk oss på [Slack#naisflow](https://nav-it.slack.com/messages/CGRMQHT50).
