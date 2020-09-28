package zookeeper

import (
	"github.com/go-zookeeper/zk"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog"
	"time"
)

var _ = Describe("Test case for creating a new PathCache", func() {
	// Connect a cluster which has been started in reality
	conn, _, err := zk.Connect(testOpt.Address, 30*time.Second)
	Expect(err).NotTo(HaveOccurred())

	// Connect a test cluster
	// os.Setenv("ZOOKEEPER_BIN_PATH", "/Users/yangyongzhi/zookeeper-3.4.14/bin")
	// ts, err := StartTestCluster(1, nil, nil)
	// if err != nil {
	//	klog.Errorf("Start a test cluster has an error: %v", err)
	// }
	// defer ts.Stop()
	// Expect(err).NotTo(HaveOccurred())
	// conn, _, err := ts.ConnectWithOptions(15 * time.Second)
	// Expect(err).NotTo(HaveOccurred())

	BeforeEach(func() {
		klog.Infof("before process for each context")
	})

	Describe("The node which was been watched has not been exist", func() {
		mockPath := "/foo"
		Context("For path foo", func() {
			It("Path foo has not been exist until now", func() {
				pc, err := NewPathCache(conn, mockPath, "Tester", false, true)
				Expect(err).NotTo(HaveOccurred())
				Expect(pc).NotTo(BeNil())

				data, stat, err := conn.Get(mockPath)
				klog.Infof("The data of Path foo %d is :%s", stat.Czxid, string(data))
				Expect(err).NotTo(HaveOccurred())
				Expect(data).To(BeNil())
			})
		})

		AfterEach(func() {
			klog.Infof("Clear all the nodes we have created")
			err := conn.Delete(mockPath, 0)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Test the create path recursively", func() {
		path := "/path/to"
		Context("", func() {
			It("", func() {
				createRecursivelyIfNecessary(conn, path)
				_, _, err = conn.Get(path)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		AfterEach(func() {
			klog.Infof("Clear all the nodes we have created")
			err := conn.Delete(path, 0)
			Expect(err).NotTo(HaveOccurred())
			err = conn.Delete("/path", 0)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Test the create path directly", func() {
		path := "/node/test"
		_, err = conn.Create("/node", nil, 0, worldACL(PermAll))
		Expect(err).NotTo(HaveOccurred())

		Context("", func() {
			It("", func() {
				create(conn, path)

				_, _, err = conn.Get(path)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		AfterEach(func() {
			klog.Infof("Clear all the nodes we have created")
			err := conn.Delete(path, 0)
			Expect(err).NotTo(HaveOccurred())
			err = conn.Delete("/node", 0)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	AfterEach(func() {
		klog.Infof("after process for each context")
	})
})
