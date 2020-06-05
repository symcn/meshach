package utils

import (
	"strconv"
	"strings"

	ptypes "github.com/gogo/protobuf/types"
	"k8s.io/klog"
)

// FormatToDNS1123 converts the object name to lower case and replaces the underscore
func FormatToDNS1123(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

// StringToDuration converts the time string to protobuf Duration
// e.g.
// 2s => ptypes.Duration{Seconds: 2}
// 20ms => ptypes.Duration{Nanos: 20000000}
// 2 => ptypes.Duration{Seconds: 2}
func StringToDuration(s string) *ptypes.Duration {
	if strings.HasSuffix(s, "ms") {
		t, err := strconv.ParseInt(strings.TrimSuffix(s, "ms"), 10, 64)
		if err != nil {
			klog.Errorf("parse %s to protobuf duration error: %v", s, err)
			t = 0
		}
		return &ptypes.Duration{Nanos: int32(t) * 1000000}
	}

	t, err := strconv.ParseInt(strings.TrimSuffix(s, "s"), 10, 64)
	if err != nil {
		klog.Errorf("parse %s to protobuf duration error: %v", s, err)
		t = 0
	}
	return &ptypes.Duration{Seconds: t}
}
