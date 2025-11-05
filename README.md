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