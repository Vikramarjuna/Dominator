package testdata

import "net"

// @grpc
type GetVmInfoRequest struct {
	IpAddress net.IP `proto:"1"`
}

// @grpc
type GetVmInfoResponse struct {
	Error  string
	VmInfo VmInfo `proto:"1"`
}

// @grpc
type VmInfo struct {
	IpAddress   net.IP `proto:"1"`
	Hostname    string `proto:"2"`
	State       uint8  `proto:"3"`
	MemoryInMiB uint64 `proto:"4"`
	MilliCPUs   uint64 `proto:"5"`
}

