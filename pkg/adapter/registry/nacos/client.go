package nacos

import (
	"sync"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	nClient naming_client.INamingClient
	lock    sync.Mutex
)
var log = logf.Log.WithName("nacos")

// NewClient ...
func NewClient() (naming_client.INamingClient, error) {
	var err error
	lock.Lock()
	defer lock.Unlock()

	if nClient == nil {
		log.Info("Starting create a new nacos client.")

		nc, e := clients.CreateNamingClient(map[string]interface{}{
			"serverConfigs": []constant.ServerConfig{
				{
					IpAddr:      "10.248.224.144",
					Port:        8848,
					ContextPath: "/nacos",
				},
			},
			"clientConfig": constant.ClientConfig{
				TimeoutMs:           10 * 1000,
				BeatInterval:        5 * 1000,
				ListenInterval:      30 * 1000,
				NotLoadCacheAtStart: true,
			},
		})

		nClient = nc
		err = e
	}
	return nClient, err

}
