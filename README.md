[![Go Report Card](https://goreportcard.com/badge/github.com/symcn/mesh-operator)](https://goreportcard.com/report/github.com/symcn/mesh-operator)
[![codecov](https://codecov.io/gh/symcn/mesh-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/symcn/mesh-operator)
[![Actions Status](https://github.com/symcn/mesh-operator/workflows/go-build/badge.svg)](https://github.com/symcn/mesh-operator/actions?query=workflow%3Ago-build)
[![Actions Status](https://github.com/symcn/mesh-operator/workflows/pre-commit/badge.svg)](https://github.com/symcn/mesh-operator/actions?query=workflow%3Apre-commit)
# mesh-operator
It is in charge of all things about implementation of Service Mesh.

What are our tasks?

1. Retrieving the data of service & instances from a registry center such as Nacos, zookeeper. 
2. Generating the istio's CRD through various data such as registry data & configurations.
3. Make MOSN be able to support serve a registry request which comes from a dubbo provider.

![architecture-diagram](./docs/architecture-diagram.png)

## Installing and running

### Build

Direct compilation:

```shell
$ make manager
```

Build docker image and push:

```shell
$ make docker-build
$ make docker-push
```

You can customize the docker repository by modifying the `IMG_ADDR` variable in the [Makefile](./Makefile).

### Deploy

Install CRD:

```shell
$ make install
```

Install CRD no validations:

```shell
$ kubectl apply -f config/crd/simple
```

Create sample Meshconfig and default Sidecar:

```shell
$ kubectl apply -f config/samples/mesh_v1alpha1_meshconfig.yaml -f config/samples/istio_default_sidecar.yaml
```

Deploy adapter and controller:

```shell
$ make deploy
```

You can customize the deployment by modifying the `config/operator` and `config/adapter` kustomize config files.

### Config

#### Istio Config

```yaml
env:
  - name: PILOT_PUSH_THROTTLE
    value: "1000"
  - name: PILOT_DEBOUNCE_AFTER
    value: 100ms
  - name: PILOT_DEBOUNCE_MAX
    value: 5s
  - name: PILOT_ENABLE_EDS_DEBOUNCE
    value: "false"
  - name: PILOT_ENABLE_SERVICEENTRY_SELECT_PODS
    value: "false"
  - name: PILOT_ENABLE_K8S_SELECT_WORKLOAD_ENTRIES
    value: "false"
  - name: PILOT_XDS_CACHE_SIZE
    value: "50000"
  - name: PILOT_ENABLE_HEADLESS_SERVICE_POD_LISTENERS
    value: "false"
```

## Getting support

- [Chinese Docs](./docs/chinese.md)

## Features
Adapter:
- Synchronizing services from a specified registry center.
- Synchronizing the customized configuration of service from a specified config center
  
#### Source
The implementing for synchronizing services:
- Zookeeper servers of a dubbo cluster (Supported)
- Nacos (Planned)

The implementing for synchronizing configs:
- The configuration stored as a independent zNode for a dubbo service
- Nacos (Planned)

#### Target

The event handler what is used for creating or updating the services with its configuration
- Creating a CR named ConfiguredService correspond to a service into a single k8s cluster (Supported)
- Creating the CR into multiple k8s clusters (Supported)

### Supported

### Planned

