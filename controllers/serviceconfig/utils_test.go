package serviceconfig

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("test utils function", func() {
	Context("test truncated", func() {
		It("truncate when string length exceeds 62", func() {
			s := make([]byte, 66)
			want := make([]byte, 62)
			for i := range s {
				s[i] = 'a'
			}
			for i := range want {
				want[i] = 'a'
			}
			result := truncated(string(s))
			Expect(result).To(Equal(string(want)))
			Expect(len(result)).To(Equal(62))
		})
	})
	Context("test map contains", func() {
		It("return true", func() {
			renameMap := map[string]string{"flag": "group"}
			obj := map[string]string{"group": "blue", "app": "aaa", "service": "bbb"}
			std := map[string]string{"flag": "blue"}
			result := mapContains(std, obj, renameMap)
			Expect(result).To(Equal(true))
		}, timeout)

		It("return true", func() {
			renameMap := map[string]string{"flag": "group"}
			obj := map[string]string{"flag": "blue", "app": "aaa", "service": "bbb"}
			std := map[string]string{"app": "aaa"}
			result := mapContains(std, obj, renameMap)
			Expect(result).To(Equal(true))
		}, timeout)

		It("return false", func() {
			renameMap := map[string]string{"flag": "group"}
			obj := map[string]string{"flag": "blue", "app": "aaa", "service": "bbb"}
			std := map[string]string{"ddd": "ccc"}
			result := mapContains(std, obj, renameMap)
			Expect(result).To(Equal(false))
		}, timeout)
	})
})
