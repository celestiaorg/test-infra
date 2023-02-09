package common

import (
	"strconv"
	"strings"
)

// Used by blocksync experiments, parse a provided list of
// full node IDs that respects the following format: idA-idB-idC
// These IDs are meant to represent the order of the full node in its GroupSeq
// Returns a map to ease-up the process of lookup.
func ParseFullNodeEntryPointsKey(key string) (map[int]int, error) {
	entryPointsIDs := make(map[int]int, 0)
	res := strings.Split(key, "-")
	for _, ID := range res {
		id, err := strconv.Atoi(ID)
		if err != nil {
			return nil, err
		}
		entryPointsIDs[id] = id
	}
	return entryPointsIDs, nil
}
