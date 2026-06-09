package filesystem

import (
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
	"github.com/Cloud-Foundations/tricorder/go/tricorder/units"
)

func (objSrv *ObjectServer) registerMetrics(
	dir *tricorder.DirectorySpec) error {
	if err := dir.RegisterMetric("referenced-object-bytes",
		&objSrv.referencedBytes,
		units.Byte,
		"bytes consumed by referenced objects"); err != nil {
		return err
	}
	if err := dir.RegisterMetric("unreferenced-object-bytes",
		&objSrv.unreferencedBytes,
		units.Byte,
		"bytes consumed by unreferenced objects"); err != nil {
		return err
	}
	if err := dir.RegisterMetric("referenced-utilization-percent",
		func() float64 {
			return objSrv.utilizationPercent(objSrv.referencedBytes)
		},
		units.None,
		"referenced object bytes as percent of filesystem capacity",
	); err != nil {
		return err
	}
	if err := dir.RegisterMetric("unreferenced-utilization-percent",
		func() float64 {
			return objSrv.utilizationPercent(objSrv.unreferencedBytes)
		},
		units.None,
		"unreferenced object bytes as percent of filesystem capacity",
	); err != nil {
		return err
	}
	return nil
}

func (objSrv *ObjectServer) utilizationPercent(bytes uint64) float64 {
	_, capacity, err := objSrv.getSpaceMetrics()
	if err != nil || capacity == 0 {
		return 0
	}
	return float64(bytes) * 100 / float64(capacity)
}
