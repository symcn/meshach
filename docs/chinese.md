# Mesh Operator

该项目主要将注册在 Kubernetes 外部的服务接入集群中，并基于 [Istio](https://istio.io/) 和 [MOSN](https://mosn.io/) 实现服务网格化。目前仅适用于注册在 [Zookeeper](https://zookeeper.apache.org/) 上的 [Dubbo](https://dubbo.apache.org/zh-cn/) 服务，后续会支持如 [Nacos](https://nacos.io/zh-cn/docs/what-is-nacos.html) 等其他注册中心。

项目包含 `mesh-adapter` 和 `mesh-controller` 两种组件。

`mesh-adapter` 主要负责适配不同的注册中心，如 `Zookeeper` 和 `Nacos`，将其中的服务、实例和配置信息统一转化为预先定义的 `CRD`，交由 `mesh-controller` 处理。

`mesh-controller` 负责处理预定义的 `CRD`，如 `ConfiguredService`、`ServiceConfig`、`ServiceAccessor` 等。`mesh-controller` 通过一定的规则，将这些 `CRD` 统一转化为 `Istio` 可理解的配置，如 `ServiceEntry`、`VirtualService`、`DestinationRule` 和 `WorkloadEntry` 等，再交由 `Pilot` 将其转化为 [xDS 协议](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol) 下发至数据面。

如图：

![architecture-diagram](./architecture-diagram.png)


## Adapter

待补充。

## Controller

### CRD 概览

我们预定义了六种 CRD，分别为 `ConfiguredService`、`ServiceConfig`、`MeshConfig`、`ServiceAccessor`、`AppMeshConfig` 和 `IstioConfig`。

#### ConfiguredService

`ConfiguredService` 保存注册中心中服务及实例的信息，通过 `ServiceEntry` 将服务注册在 `Pilot` 中，服务实例通过 `WorkloadEntry` 映射。如服务实例 `label` 中包含蓝绿灰等分组信息，将创建默认的 `VirtualService` 和 `DestinationRule` 策略，`Consumer` 只能调用同组的 `Provider` 。

根据 zk 中的数据实时更新。

#### ServiceConfig

`ServiceConfig` 保存配置中心各服务及实例的配置，如重试、超时、权重、负载均衡策略等。包含动态路由，Controller 将根据此信息动态调整各个 `Subset` 的流量权重。仅更新 `VirtualService`、`DestinationRule` 和 `WorkloadEntry` 三种配置。

#### ServiceAccessor

`ServiceAccessor` 保存了同一个应用（通过标签 `app` 选择）下所有服务的调用关系，`Controller` 将根据这些调用关系生成 `Sidecar` 限定服务的作用域，通过 `egress` 限定代理可以访问的服务，当 xDS 更新时，仅下发到此作用域内的代理 MOSN 中，极大提高 xDS 下发的效率。

#### MeshConfig

`MeshConfig` 包含整个网格的元信息，如各个 `Subset` 如何设置，全局的超时、重试策略，用来筛选实例的具体标签等等。`MeshConfig` 的修改将会触发全局所有 `ConfiguredService` 、`ServiceConfig` 和 `ServiceAccessor` 的调和过程。

#### AppMeshConfig

`AppMeshConfig` 将同一应用所有服务的所有配置统一起来，主要用于统计和观测。未完善。

#### IstioConfig

`IstioConfig` 主要用来在多个集群中可定制的、自动化的安装 `Istio`，暂未实现。
