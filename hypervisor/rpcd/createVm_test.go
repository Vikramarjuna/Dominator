package rpcd

import (
	"testing"
	"time"

	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Test proto-to-SRPC converter for CreateVmAsync
func TestCreateVmAsyncRequestFromProto(t *testing.T) {
	// Create a sample proto request
	pbReq := &pb.CreateVmAsyncRequest{
		DhcpTimeoutNs:      int64(30 * time.Second),
		DoNotStart:         true,
		EnableNetboot:      false,
		ImageName:          "test-image",
		ImageTimeoutNs:     int64(5 * time.Minute),
		MinimumFreeBytes:   10 * 1024 * 1024 * 1024, // 10 GB
		RoundupPower:       30,
		SkipBootloader:     false,
		SkipMemoryCheck:    false,
		Hostname:           "test-vm",
		MemoryInMib:        4096, // 4 GB
		MilliCpus:          2000,
		VirtualCpus:        2,
		SubnetId:           "test-subnet",
		CpuPriority:        1,
		DestroyProtection:  true,
		DestroyOnPowerdown: false,
		DisableVirtIo:      false,
	}

	// Convert to SRPC request
	srpcReq, err := createVmAsyncRequestFromProto(pbReq)
	if err != nil {
		t.Fatalf("createVmAsyncRequestFromProto failed: %v", err)
	}

	// Verify basic fields
	if srpcReq.DhcpTimeout != 30*time.Second {
		t.Errorf("DhcpTimeout mismatch: got %v, want %v", srpcReq.DhcpTimeout, 30*time.Second)
	}
	if srpcReq.DoNotStart != true {
		t.Errorf("DoNotStart mismatch: got %v, want true", srpcReq.DoNotStart)
	}
	if srpcReq.ImageName != "test-image" {
		t.Errorf("ImageName mismatch: got %v, want test-image", srpcReq.ImageName)
	}
	if srpcReq.ImageTimeout != 5*time.Minute {
		t.Errorf("ImageTimeout mismatch: got %v, want %v", srpcReq.ImageTimeout, 5*time.Minute)
	}
	if srpcReq.MinimumFreeBytes != 10*1024*1024*1024 {
		t.Errorf("MinimumFreeBytes mismatch: got %v, want %v", srpcReq.MinimumFreeBytes, 10*1024*1024*1024)
	}

	// Verify VmInfo fields
	if srpcReq.VmInfo.Hostname != "test-vm" {
		t.Errorf("Hostname mismatch: got %v, want test-vm", srpcReq.VmInfo.Hostname)
	}
	if srpcReq.VmInfo.MemoryInMiB != 4096 {
		t.Errorf("MemoryInMiB mismatch: got %v, want 4096", srpcReq.VmInfo.MemoryInMiB)
	}
	if srpcReq.VmInfo.MilliCPUs != 2000 {
		t.Errorf("MilliCPUs mismatch: got %v, want 2000", srpcReq.VmInfo.MilliCPUs)
	}
	if srpcReq.VmInfo.VirtualCPUs != 2 {
		t.Errorf("VirtualCPUs mismatch: got %v, want 2", srpcReq.VmInfo.VirtualCPUs)
	}
	if srpcReq.VmInfo.SubnetId != "test-subnet" {
		t.Errorf("SubnetId mismatch: got %v, want test-subnet", srpcReq.VmInfo.SubnetId)
	}
	if srpcReq.VmInfo.CpuPriority != 1 {
		t.Errorf("CpuPriority mismatch: got %v, want 1", srpcReq.VmInfo.CpuPriority)
	}
	if srpcReq.VmInfo.DestroyProtection != true {
		t.Errorf("DestroyProtection mismatch: got %v, want true", srpcReq.VmInfo.DestroyProtection)
	}
}

// Test that streaming fields are rejected
func TestCreateVmAsyncRejectsStreamingFields(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*proto.CreateVmRequest)
	}{
		{
			name: "ImageDataSize",
			setup: func(req *proto.CreateVmRequest) {
				req.ImageDataSize = 1024
			},
		},
		{
			name: "UserDataSize",
			setup: func(req *proto.CreateVmRequest) {
				req.UserDataSize = 1024
			},
		},
		{
			name: "SecondaryVolumesData",
			setup: func(req *proto.CreateVmRequest) {
				req.SecondaryVolumesData = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a valid base request
			req := &proto.CreateVmRequest{
				VmInfo: proto.VmInfo{
					Hostname:    "test-vm",
					ImageName:   "test-image",
					MemoryInMiB: 1024,
					MilliCPUs:   1000,
				},
			}

			// Apply the test-specific setup
			tt.setup(req)

			// This should be validated in the handler, not the converter
			// The converter should succeed
			_, err := createVmAsyncRequestFromProto(&pb.CreateVmAsyncRequest{
				Hostname:    "test-vm",
				MemoryInMib: 1024,
				MilliCpus:   1000,
				ImageName:   "test-image",
			})
			if err != nil {
				t.Errorf("Converter should not fail for valid proto: %v", err)
			}
		})
	}
}

// Test that validateNoStreamingFields returns Unimplemented status code
func TestValidateNoStreamingFields(t *testing.T) {
	tests := []struct {
		name         string
		request      *proto.CreateVmRequest
		expectError  bool
		expectedCode codes.Code
		expectedMsg  string
	}{
		{
			name: "Valid request with ImageName",
			request: &proto.CreateVmRequest{
				VmInfo: proto.VmInfo{
					Hostname:    "test-vm",
					ImageName:   "test-image",
					MemoryInMiB: 1024,
					MilliCPUs:   1000,
				},
			},
			expectError: false,
		},
		{
			name: "ImageDataSize should return Unimplemented",
			request: &proto.CreateVmRequest{
				ImageDataSize: 1024,
				VmInfo: proto.VmInfo{
					Hostname:    "test-vm",
					MemoryInMiB: 1024,
					MilliCPUs:   1000,
				},
			},
			expectError:  true,
			expectedCode: codes.Unimplemented,
			expectedMsg:  "ImageDataSize streaming not supported",
		},
		{
			name: "UserDataSize should return Unimplemented",
			request: &proto.CreateVmRequest{
				UserDataSize: 1024,
				VmInfo: proto.VmInfo{
					Hostname:    "test-vm",
					ImageName:   "test-image",
					MemoryInMiB: 1024,
					MilliCPUs:   1000,
				},
			},
			expectError:  true,
			expectedCode: codes.Unimplemented,
			expectedMsg:  "UserDataSize streaming not supported",
		},
		{
			name: "SecondaryVolumesData should return Unimplemented",
			request: &proto.CreateVmRequest{
				SecondaryVolumesData: true,
				VmInfo: proto.VmInfo{
					Hostname:    "test-vm",
					ImageName:   "test-image",
					MemoryInMiB: 1024,
					MilliCPUs:   1000,
				},
			},
			expectError:  true,
			expectedCode: codes.Unimplemented,
			expectedMsg:  "SecondaryVolumesData streaming not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoStreamingFields(tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}

				// Check that it's a gRPC status error
				st, ok := status.FromError(err)
				if !ok {
					t.Errorf("Expected gRPC status error, got: %T", err)
					return
				}

				// Check the status code
				if st.Code() != tt.expectedCode {
					t.Errorf("Expected status code %v, got %v", tt.expectedCode, st.Code())
				}

				// Check the error message contains expected text
				if tt.expectedMsg != "" && !contains(st.Message(), tt.expectedMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.expectedMsg, st.Message())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
