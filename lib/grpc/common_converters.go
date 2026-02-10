package grpc

import (
	"net"

	common_grpc "github.com/Cloud-Foundations/Dominator/proto/common/grpc"
	"github.com/Cloud-Foundations/Dominator/lib/tags"
)

// Common converters for proto/common/grpc types.
// These converters are used across multiple services to convert between
// gRPC proto types and internal types.

// TagsFromProto converts protobuf Tags to internal tags.Tags.
// This mirrors lib/tags.Tags (map[string]string).
func TagsFromProto(pb *common_grpc.Tags) tags.Tags {
	if pb == nil || pb.Tags == nil {
		return nil
	}
	return tags.Tags(pb.Tags)
}

// TagsToProto converts internal tags.Tags to protobuf Tags.
func TagsToProto(t tags.Tags) *common_grpc.Tags {
	if t == nil {
		return nil
	}
	return &common_grpc.Tags{Tags: t}
}

// MatchTagsFromProto converts protobuf MatchTags to internal tags.MatchTags.
// This mirrors lib/tags.MatchTags (map[string][]string).
func MatchTagsFromProto(pb *common_grpc.MatchTags) tags.MatchTags {
	if pb == nil || pb.MatchTags == nil {
		return nil
	}
	result := make(tags.MatchTags)
	for k, v := range pb.MatchTags {
		if v != nil {
			result[k] = v.Values
		}
	}
	return result
}

// MatchTagsToProto converts internal tags.MatchTags to protobuf MatchTags.
func MatchTagsToProto(mt tags.MatchTags) *common_grpc.MatchTags {
	if mt == nil {
		return nil
	}
	result := &common_grpc.MatchTags{
		MatchTags: make(map[string]*common_grpc.StringList),
	}
	for k, v := range mt {
		result.MatchTags[k] = &common_grpc.StringList{Values: v}
	}
	return result
}

// IpFromBytes converts a byte slice to net.IP.
// This is a common helper used by gRPC handlers.
func IpFromBytes(b []byte) net.IP {
	return net.IP(b)
}

// ErrorToString converts an error to a string, returning empty string for nil.
// DEPRECATED: Use ErrorToStatus() instead for proper gRPC error handling.
// This function is kept for backward compatibility during migration.
func ErrorToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

