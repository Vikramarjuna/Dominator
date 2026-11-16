package convtypes

import (
	"net"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/tags"
)

type Blob []byte

type Labels tags.Tags

type Level uint

type Moment time.Time

type Name string

type Names []Name

type Ref *Inner

type Inner struct {
	Count int16
	Name  string
}

// @grpc
type Everything struct {
	Addresses []net.IP
	Blob      Blob
	ByLevel   map[Level]string
	ByName    map[Name]Name
	Created   time.Time
	Hardware  net.HardwareAddr
	Inner     Inner
	Inners    []Inner
	Labels    Labels
	Level     Level
	Levels    []Level
	Moment    Moment
	Names     Names
	Pointer   *Inner
	Raw       []byte
	Ref       Ref
	Timeout   time.Duration
	Title     Name
	Times     []time.Time
}
