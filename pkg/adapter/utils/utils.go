package utils

import (
	"net"
	"path"
	"reflect"
	"strconv"
	"strings"

	"k8s.io/klog"
)

// DeleteInSlice delete an element from a Slice with an index.
// return the original parameter as the result instead if it is not a slice.
func DeleteInSlice(s interface{}, index int) interface{} {
	value := reflect.ValueOf(s)
	if value.Kind() == reflect.Slice {
		//|| value.Kind() == reflect.Array {
		result := reflect.AppendSlice(value.Slice(0, index), value.Slice(index+1, value.Len()))
		return result.Interface()
	}

	klog.Errorf("Only a slice can be passed into this method for deleting an element of it.")
	return s
}

// RemovePort removing the port part of a service name is necessary due to istio requirement.
// 127.0.0.1:10000 -> 127.0.0.1
func RemovePort(addressWithPort string) string {
	host, _, err := net.SplitHostPort(addressWithPort)
	if err != nil {
		klog.Errorf("Split host and port for a service name has an error:%v\n", err)
		// returning the original address instead if the address has a incorrect format
		return addressWithPort
	}
	return host
}

// ResolveServiceName ...
// configuratorPath: e.g. /dubbo/config/dubbo/foo.configurators
func ResolveServiceName(configuratorPath string) string {
	return strings.Replace(path.Base(configuratorPath), ".configurators", "", 1)
}

// ToUint32 Convert a string variable to integer with 32 bit size.
func ToUint32(portStr string) uint32 {
	port, _ := strconv.ParseInt(portStr, 10, 32)
	return uint32(port)
}

// ToInt32 Convert a string variable to integer with 32 bit size.
func ToInt32(portStr string) int32 {
	port, _ := strconv.ParseInt(portStr, 10, 32)
	return int32(port)
}

// StandardizeServiceName ...
func StandardizeServiceName(originName string) string {
	return strings.ToLower(strings.ReplaceAll(originName, "-", "_"))
}
