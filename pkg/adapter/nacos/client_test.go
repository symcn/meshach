package nacos

import (
	"fmt"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

var headers = map[string][]string{
	"Client-Version":  {constant.CLIENT_VERSION},
	"User-Agent":      {constant.CLIENT_VERSION},
	"Accept-Encoding": {"gzip,deflate,sdch"},
	"Connection":      {"Keep-Alive"},
	"Request-Module":  {"Naming"},
	"Content-Type":    {"application/x-www-form-urlencoded"},
}

func Test_RegisterServiceInstance(t *testing.T) {
	client, _ := NewClient()

	// test case for singleton variable
	// NewClient()

	success, err := client.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: "DEMO",
		Ip:          "10.0.0.10",
		Port:        80,
		Ephemeral:   true,
		Enable:      true, // you should pass a value for these field due to zero value mechanism
		Weight:      1,
		Metadata:    map[string]string{"foo": "bar"},
	})

	assert.Equal(t, nil, err)
	assert.Equal(t, true, success)

	time.Sleep(30 * time.Minute)
}

func TestNamingProxy_GetService_WithoutGroupName(t *testing.T) {
	client, _ := NewClient()
	result, err := client.GetService(vo.GetServiceParam{
		ServiceName: "DEMO",
		//Clusters:    []string{"a"},
	})
	assert.Nil(t, err)
	fmt.Printf("Services : %v\n", result)
	//assert.Equal(t, serviceTest, result)

}

func TestNamingClient_SelectAllInstances(t *testing.T) {
	client, _ := NewClient()
	instances, err := client.SelectAllInstances(vo.SelectAllInstancesParam{
		ServiceName: "DEMO",
		//Clusters:    []string{"DEFAULT"},
	})
	fmt.Println(utils.ToJsonString(instances))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(instances))
}

func TestNamingClient_GetAllServicesInfo(t *testing.T) {
	client, _ := NewClient()
	serviceInfos, err := client.GetAllServicesInfo(vo.GetAllServiceInfoParam{
		Clusters: []string{"DEFAULT"},
	})
	fmt.Println(utils.ToJsonString(serviceInfos))
	assert.Nil(t, err)
	assert.Equal(t, 1, len(serviceInfos))
}

func TestNamingClient_Subscribe(t *testing.T) {
	client, _ := NewClient()
	err := client.Subscribe(&vo.SubscribeParam{
		Clusters:    []string{"DEFAULT"},
		ServiceName: "DEMO",
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			// assert.Equal(t, 0, services)
			fmt.Println(utils.ToJsonString(services))
		},
	})

	assert.Nil(t, err)

	time.Sleep(time.Minute)
}
