package main

import "github.com/shirou/gopsutil/v4/mem"

type vmStat struct {
	Total     uint64
	Used      uint64
	Available uint64
}

func getVirtualMemory() (vmStat, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return vmStat{}, err
	}
	return vmStat{
		Total:     v.Total,
		Used:      v.Used,
		Available: v.Available,
	}, nil
}
