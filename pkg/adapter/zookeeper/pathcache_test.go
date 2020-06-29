package zookeeper

// func Test_watch(t *testing.T) {
// 	conn, _, err := zk.Connect(ZkServers, time.Duration(15*time.Hour))
// 	if err != nil {
// 		fmt.Printf("connect to zookeeper has an err: %v\n", err)
// 		return
// 	}

// 	//go func() {
// 	p, err := newPathCache(conn, DubboRootPath, "MOCK", true)
// 	if err != nil {
// 		fmt.Printf("Create a new pathcache for [%s] has an err: %v\n", DubboRootPath, err)
// 		assert.EqualError(t, err, "zk: node does not exist")
// 		return
// 	}
// 	fmt.Printf("Created a path cache : %s\n", p.path)
// 	//}()

// 	time.Sleep(30 * time.Minute)

// }
