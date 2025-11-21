# function-gitlab-importer
[![CI](https://github.com/simon-fredrich/function-gitlab-importer/actions/workflows/ci.yml/badge.svg)](https://github.com/simon-fredrich/function-gitlab-importer/actions/workflows/ci.yml)

A Function for importing existing gitlab resources into crossplane.

## Getting Started
To get started create the following file to use the function locally.
```yaml
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-gitlab-importer
  annotations:
    # This tells crossplane beta render to connect to the function locally.
    render.crossplane.io/runtime: Development
spec:
  # This is ignored when using the Development runtime.
  package: ghcr.io/simon-fredrich/function-gitlab-importer:<tag>
```
## Setup Github-Credentials
There are two possible ways to provide the function with the gitlab-api-token.
### As Input
`token` and `baseUrl` can be specified on a per-call-level.
```yaml
- step: run-function
  functionRef:
    name: function-gitlab-importer
  input:
    token: <gitlab-api-token>
    baseUrl: <gitlab-baseUrl>
```
### As Environment-Variable
The gitlab-credentials can also be specified as environment-variables within the container running the function. For that you have to specify the following variables.
```shell
$ export GITLAB_API_TOKEN=<gitlab-api-token>
$ export GITLAB_URL=<gitlab_url>
```
## Run Function Locally
Open a terminal and run the following command in the project directory.
```shell
$ go run . --insecure --debug
```
To test the function one might need additional resources. These can be provided in the folder `example/observed` and used withing the rendering call. Open a second terminal, navigate to the your resource definitions, and run the following command:
```shell
$ crossplane render \
  --observed-resources /observed \
  --include-full-xr \
  --include-context \
  xr.yaml composition.yaml functions.yaml
```
## Operate Function in Production
Setup a DeploymentRuntimeConfig alongside your your function.
```yaml
---
apiVersion: pkg.crossplane.io/v1beta1
kind: Function
metadata:
  name: function-gitlab-importer
spec:
  package: ghcr.io/simon-fredrich/function-gitlab-importer:<tag>
  runtimeConfigRef:
    name: gitlab-credentials-config
---
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: gitlab-credentials-config
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              env:
                - name: GITLAB_API_KEY
                  valueFrom:
                    secretKeyRef:
                      key: token
                      name: gitlab-credentials
                - name: GITLAB_URL
                  value: https://gitlab.com/
```