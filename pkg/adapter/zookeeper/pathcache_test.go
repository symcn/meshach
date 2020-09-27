package zookeeper

import (
	"github.com/go-zookeeper/zk"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/klog"
	"time"
)

var _ = Describe("Test case for creating a new PathCache", func() {
	conn, _, err := zk.Connect(testOpt.Address, 30*time.Second)
	Expect(err).NotTo(HaveOccurred())

	BeforeEach(func() {
		klog.Infof("before process for each context")
	})

	Describe("The node which was been watched has not been exist", func() {
		mockPath := "/foo/bar"
		Context("For path foo", func() {
			It("Path foo has not been exist until now", func() {
				pc, err := NewPathCache(conn, mockPath, "Tester", false, true)
				Expect(err).NotTo(HaveOccurred())
				Expect(pc).NotTo(BeNil())

				data, stat, err := conn.Get(mockPath)
				klog.Infof("The data of Path foo %d is :%s", stat.Czxid, string(data))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		AfterEach(func() {
			klog.Infof("Clear all the nodes we created")
			err := conn.Delete(mockPath, 0)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	AfterEach(func() {
		klog.Infof("after process for each context")
	})
})
