package common

import "fmt"

func GetBandwidthValue(v string) uint64 {
	var bandwidthMap = map[string]uint64{
		"100Mib":  13 << 23,
		"256Mib":  4 << 26,
		"320Mib":  5 << 26,
		"512Mib":  8 << 26,
		"1024Mib": 16 << 26,
	}
	if val, ok := bandwidthMap[v]; ok {
		return val
	} else {
		panic(fmt.Errorf("can't find any bandwidth value to given - %s", v))
	}
}
