package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/clients/nacos_client"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/common/http_agent"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
)

var (
	nClient *naming_client.NamingClient
	lock    sync.Mutex
)
var log = logf.Log.WithName("nacos")

func NewClient() (*naming_client.NamingClient, error) {
	var err error
	lock.Lock()
	defer lock.Unlock()

	if nClient == nil {
		log.Info("Starting create a new nacos client.")

		clientConfig := constant.ClientConfig{
			TimeoutMs:           10 * 1000,
			BeatInterval:        5 * 1000,
			ListenInterval:      30 * 1000,
			NotLoadCacheAtStart: true,
		}

		var serverConfig = constant.ServerConfig{
			IpAddr:      "10.248.224.144",
			Port:        8848,
			ContextPath: "/nacos",
		}

		nc := nacos_client.NacosClient{}
		nc.SetServerConfig([]constant.ServerConfig{serverConfig})
		nc.SetClientConfig(clientConfig)
		nc.SetHttpAgent(&http_agent.HttpAgent{})
		c, e := naming_client.NewNamingClient(&nc)
		nClient = &c
		err = e
	}
	return nClient, err

}
