[![Actions Status](https://github.com/symcn/mesh-operator/workflows/go-build/badge.svg)](https://github.com/symcn/mesh-operator/actions?query=workflow%3Ago-build)
[![Actions Status](https://github.com/symcn/mesh-operator/workflows/pre-commit/badge.svg)](https://github.com/symcn/mesh-operator/actions?query=workflow%3Apre-commit)
# mesh-operator
It is in charge of all things about implementation of Service Mesh.

What are our tasks?

1. Retrieving the data of service & instances from a registry center such as Nacos, zookeeper. 
2. Generating the istio's CRD through various data such as registry data & configurations.
3. Make MOSN can support registry request of a dubbo provider.


## Unit Test

```shell
➜ make test-controller
go test -v -cover github.com/symcn/mesh-operator/pkg/controller/appmeshconfig
=== RUN   TestReconcileAppMeshConfig_Reconcile
=== RUN   TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-no-service-ok
I0618 14:32:33.097190   29000 controller.go:138] Reconciling AppMeshConfig: sym-test/amc-test-case
I0618 14:32:33.100031   29000 controller.go:181] Update AppMeshConfig[sym-test/amc-test-case] status...
I0618 14:32:33.100830   29000 controller.go:188] End Reconciliation, AppMeshConfig: sym-test/amc-test-case.
=== RUN   TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-no-meshconfig-error
I0618 14:32:33.100902   29000 controller.go:138] Reconciling AppMeshConfig: sym-test/amc-test-case
E0618 14:32:33.100913   29000 controller.go:144] Get cluster MeshConfig[sym-test/mc-test-case] error: meshconfigs.mesh.symcn.com "mc-test-case" not found
=== RUN   TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-no-appmeshconfig-ok
I0618 14:32:33.100944   29000 controller.go:138] Reconciling AppMeshConfig: sym-test/amc-test-case
I0618 14:32:33.101047   29000 controller.go:157] Can't found AppMeshConfig[sym-test/amc-test-case], requeue...
=== RUN   TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-only-serviceentry-ok
I0618 14:32:33.101078   29000 controller.go:138] Reconciling AppMeshConfig: sym-test/amc-test-case
I0618 14:32:33.101241   29000 serviceentry.go:51] Creating a new ServiceEntry, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.102085   29000 controller.go:181] Update AppMeshConfig[sym-test/amc-test-case] status...
I0618 14:32:33.102326   29000 controller.go:188] End Reconciliation, AppMeshConfig: sym-test/amc-test-case.
=== RUN   TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-only-workloadentry-ok
I0618 14:32:33.102365   29000 controller.go:138] Reconciling AppMeshConfig: sym-test/amc-test-case
I0618 14:32:33.102522   29000 workloadentry.go:55] Creating a new WorkloadEntry, Namespace: sym-test, Name: dubbo.testserviceok.10.10.10.10.20882
I0618 14:32:33.102574   29000 serviceentry.go:51] Creating a new ServiceEntry, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.102627   29000 controller.go:181] Update AppMeshConfig[sym-test/amc-test-case] status...
I0618 14:32:33.102937   29000 controller.go:188] End Reconciliation, AppMeshConfig: sym-test/amc-test-case.
=== RUN   TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-all-ok
I0618 14:32:33.102984   29000 controller.go:138] Reconciling AppMeshConfig: sym-test/amc-test-case
I0618 14:32:33.103166   29000 workloadentry.go:55] Creating a new WorkloadEntry, Namespace: sym-test, Name: dubbo.testserviceok.10.10.10.10.20882
I0618 14:32:33.103220   29000 serviceentry.go:51] Creating a new ServiceEntry, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.103258   29000 destinationrule.go:64] Creating a new DestinationRule, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.103508   29000 virtualservice.go:51] Creating a new VirtualService, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.103741   29000 controller.go:181] Update AppMeshConfig[sym-test/amc-test-case] status...
I0618 14:32:33.104493   29000 controller.go:188] End Reconciliation, AppMeshConfig: sym-test/amc-test-case.
--- PASS: TestReconcileAppMeshConfig_Reconcile (0.01s)
    --- PASS: TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-no-service-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-no-meshconfig-error (0.00s)
    --- PASS: TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-no-appmeshconfig-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-only-serviceentry-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-only-workloadentry-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_Reconcile/test-amc-reconcile-all-ok (0.00s)
=== RUN   TestReconcileAppMeshConfig_reconcileDestinationRule
=== RUN   TestReconcileAppMeshConfig_reconcileDestinationRule/test-reconcile-destination-create-ok
I0618 14:32:33.104657   29000 destinationrule.go:64] Creating a new DestinationRule, Namespace: sym-test, Name: dubbo.testserviceok
=== RUN   TestReconcileAppMeshConfig_reconcileDestinationRule/test-reconcile-destination-update-ok
I0618 14:32:33.104777   29000 destinationrule.go:74] Update DestinationRule, Namespace: sym-test, Name: dubbo.testserviceok
=== RUN   TestReconcileAppMeshConfig_reconcileDestinationRule/test-reconcile-destination-delete-ok
I0618 14:32:33.104907   29000 destinationrule.go:74] Update DestinationRule, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.104924   29000 destinationrule.go:100] Delete unused DestinationRule: dubbo.testservice.delete
--- PASS: TestReconcileAppMeshConfig_reconcileDestinationRule (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileDestinationRule/test-reconcile-destination-create-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileDestinationRule/test-reconcile-destination-update-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileDestinationRule/test-reconcile-destination-delete-ok (0.00s)
=== RUN   TestReconcileAppMeshConfig_reconcileServiceEntry
=== RUN   TestReconcileAppMeshConfig_reconcileServiceEntry/test-reconcile-serviceentry-create-ok
I0618 14:32:33.105054   29000 serviceentry.go:51] Creating a new ServiceEntry, Namespace: sym-test, Name: dubbo.testserviceok
=== RUN   TestReconcileAppMeshConfig_reconcileServiceEntry/test-reconcile-serviceentry-update-ok
I0618 14:32:33.105192   29000 serviceentry.go:60] Update ServiceEntry, Namespace: sym-test, Name: dubbo.testserviceok
=== RUN   TestReconcileAppMeshConfig_reconcileServiceEntry/test-reconcile-serviceentry-delete-ok
I0618 14:32:33.105354   29000 serviceentry.go:60] Update ServiceEntry, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.105368   29000 serviceentry.go:84] Delete unused ServiceEntry: dubbo.testserviceok.delete
--- PASS: TestReconcileAppMeshConfig_reconcileServiceEntry (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileServiceEntry/test-reconcile-serviceentry-create-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileServiceEntry/test-reconcile-serviceentry-update-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileServiceEntry/test-reconcile-serviceentry-delete-ok (0.00s)
=== RUN   TestReconcileAppMeshConfig_updateStatus
=== RUN   TestReconcileAppMeshConfig_updateStatus/test-status-update-ok
I0618 14:32:33.105725   29000 status_test.go:126] [serviceentry] desired: 1, distributed: 1, undistributed: 0
I0618 14:32:33.105733   29000 status_test.go:131] [workloadentry] desired: 1, distributed: 1, undistributed: 0
I0618 14:32:33.105737   29000 status_test.go:136] [virtualservice] desired: 1, distributed: 1, undistributed: 0
I0618 14:32:33.105741   29000 status_test.go:141] [destinationrule] desired: 1, distributed: 1, undistributed: 0
=== RUN   TestReconcileAppMeshConfig_updateStatus/test-status-update-error
I0618 14:32:33.105839   29000 status_test.go:126] [serviceentry] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.105845   29000 status_test.go:131] [workloadentry] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.105848   29000 status_test.go:136] [virtualservice] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.105851   29000 status_test.go:141] [destinationrule] desired: 1, distributed: 0, undistributed: 1
=== RUN   TestReconcileAppMeshConfig_updateStatus/test-status-update-distributing
I0618 14:32:33.106059   29000 status_test.go:126] [serviceentry] desired: 1, distributed: 1, undistributed: 0
I0618 14:32:33.106066   29000 status_test.go:131] [workloadentry] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.106069   29000 status_test.go:136] [virtualservice] desired: 1, distributed: 1, undistributed: 0
I0618 14:32:33.106073   29000 status_test.go:141] [destinationrule] desired: 1, distributed: 0, undistributed: 1
=== RUN   TestReconcileAppMeshConfig_updateStatus/test-status-update-undistributed
I0618 14:32:33.106187   29000 status_test.go:126] [serviceentry] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.106194   29000 status_test.go:131] [workloadentry] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.106197   29000 status_test.go:136] [virtualservice] desired: 1, distributed: 0, undistributed: 1
I0618 14:32:33.106201   29000 status_test.go:141] [destinationrule] desired: 1, distributed: 0, undistributed: 1
--- PASS: TestReconcileAppMeshConfig_updateStatus (0.00s)
    --- PASS: TestReconcileAppMeshConfig_updateStatus/test-status-update-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_updateStatus/test-status-update-error (0.00s)
    --- PASS: TestReconcileAppMeshConfig_updateStatus/test-status-update-distributing (0.00s)
    --- PASS: TestReconcileAppMeshConfig_updateStatus/test-status-update-undistributed (0.00s)
=== RUN   Test_calcPhase
=== RUN   Test_calcPhase/test-calc-phase-status-unkonwn-ok
--- PASS: Test_calcPhase (0.00s)
    --- PASS: Test_calcPhase/test-calc-phase-status-unkonwn-ok (0.00s)
=== RUN   TestReconcileAppMeshConfig_reconcileVirtualService
=== RUN   TestReconcileAppMeshConfig_reconcileVirtualService/test-reconcile-virtualservice-create-ok
I0618 14:32:33.106410   29000 virtualservice.go:51] Creating a new VirtualService, Namespace: sym-test, Name: dubbo.testserviceok
=== RUN   TestReconcileAppMeshConfig_reconcileVirtualService/test-reconcile-virtualservice-update-ok
I0618 14:32:33.106536   29000 virtualservice.go:61] Update VirtualService, Namespace: sym-test, Name: dubbo.testserviceok
=== RUN   TestReconcileAppMeshConfig_reconcileVirtualService/test-reconcile-virtualservice-delete-ok
I0618 14:32:33.106708   29000 virtualservice.go:61] Update VirtualService, Namespace: sym-test, Name: dubbo.testserviceok
I0618 14:32:33.106730   29000 virtualservice.go:87] Delete unused VirtualService: dubbo.testserviceok.delete
--- PASS: TestReconcileAppMeshConfig_reconcileVirtualService (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileVirtualService/test-reconcile-virtualservice-create-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileVirtualService/test-reconcile-virtualservice-update-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileVirtualService/test-reconcile-virtualservice-delete-ok (0.00s)
=== RUN   TestReconcileAppMeshConfig_reconcileWorkloadEntry
=== RUN   TestReconcileAppMeshConfig_reconcileWorkloadEntry/test-reconcile-workloadentry-create-ok
I0618 14:32:33.106953   29000 workloadentry.go:55] Creating a new WorkloadEntry, Namespace: sym-test, Name: dubbo.testserviceok.10.10.10.10.20882
=== RUN   TestReconcileAppMeshConfig_reconcileWorkloadEntry/test-reconcile-workloadentry-update-ok
I0618 14:32:33.107141   29000 workloadentry.go:67] Update WorkloadEntry, Namespace: sym-test, Name: dubbo.testserviceok.10.10.10.10.20882
=== RUN   TestReconcileAppMeshConfig_reconcileWorkloadEntry/test-reconcile-workloadentry-delete-ok
I0618 14:32:33.107288   29000 workloadentry.go:67] Update WorkloadEntry, Namespace: sym-test, Name: dubbo.testserviceok.10.10.10.10.20882
I0618 14:32:33.107309   29000 workloadentry.go:92] Delete unused WorkloadEntry: dubbo.testserviceok.10.10.10.11.20882
--- PASS: TestReconcileAppMeshConfig_reconcileWorkloadEntry (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileWorkloadEntry/test-reconcile-workloadentry-create-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileWorkloadEntry/test-reconcile-workloadentry-update-ok (0.00s)
    --- PASS: TestReconcileAppMeshConfig_reconcileWorkloadEntry/test-reconcile-workloadentry-delete-ok (0.00s)
PASS
coverage: 72.7% of statements
ok  	github.com/symcn/mesh-operator/pkg/controller/appmeshconfig	0.059s	coverage: 72.7% of statements
```
