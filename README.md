# mesh-operator
It is in charge of all things about implementation of Service Mesh.

What are our tasks?

1. Retrieving the data of service & instances from a registry center such as Nacos, zookeeper. 
2. Generating the istio's CRD through various data such as registry data & configurations.
3. Make MOSN can support registry request of a dubbo provider.
