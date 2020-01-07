Mutatingflow
============

> A mutating webhook for adding Nais-specific dependencies to Notebooks created by Kubeflow

Foreløpig er hele prosessen med utvikling, release, og deployment manuelt. Nedenfor finner man en semi-generell, og litt NAIS spesifikk deployment-pipeline. For nå så bruker vi [nais-yaml](https://github.com/navikt/nais-yaml/)-repoet for deployment av nye versjoner.


## Development and creating new release

1. Code code code
2. `make release docker`
3. Se *Deploy Mutatingflow*

### Oppdatering av Notebooks apiet

Vi henter Notebooks (pkg/apis/notebook/v1alpha/types.go) fra [Kubeflow]()-gitrepo'et. Vi trenger ikke noe kodegenerering, da vi kun er interresert i struct'en til Notebooks.


## Deploy Mutatingflow

Hvis det ikke er første gang, hopp rett til punkt 4.

1. `kubectl apply -f deployment/secret.yaml`
2. `kubectl apply -f deployment/service.yaml`
3. `kubectl apply -f deployment/webhook.yaml`
4. Oppdatert `spec.template.spec.containers[0].image` i filen `deployment/deployment.yaml`
5. `kubectl apply -f deployment/deployment.yaml`

### Nye sertifikater

Du finner root ca/key i Vault.

Start med å generer certificate key:
```
openssl genrsa -out mutatingflow.key 2048
```

Så en one-line kommando for csr:
```
openssl req -new -sha256 -key mutatingflow.key -subj "/C=NO/ST=Oslo/O=NAV/OU=Aura/CN=mutatingflow.kubeflow.svc" -out mutatingflow.csr
```

PS: `CN` verdien bestemmer hvor tjenesten kan ligge. `mutatingflow.kubeflow.svc` betyr at det er en `service` som heter
`mutatingflow` i `kubeflow` namespacet.

Deretter kan du lage en key:
```
openssl x509 -req -in mutatingflow.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out mutatingflow.crt -days 500 -sha256
```

Oppskrift hentet fra [self-signed-certificate-with-custom-ca.md](https://gist.github.com/fntlnz/cf14feb5a46b2eda428e000157447309).

Så kan nye `mutatingflow.crt` og `mutatingflow.key` base64-encodes og legges inn `kubeflow-yaml/vars/mutatingflow.yaml`.

## Support

Besøk oss på [Slack#naisflow](https://nav-it.slack.com/messages/CGRMQHT50).
