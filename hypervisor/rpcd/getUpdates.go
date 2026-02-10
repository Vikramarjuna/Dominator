package rpcd

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
	pb "github.com/Cloud-Foundations/Dominator/proto/hypervisor/grpc"
)

const (
	flushDelay     = time.Millisecond * 10
	heartbeatDelay = time.Minute * 15
)

func (t *srpcType) GetUpdates(conn *srpc.Conn) error {
	heartbeatTimer := time.NewTimer(heartbeatDelay)
	closeChannel, responseChannel := t.getUpdatesReader(conn, heartbeatTimer)
	updateChannel := t.manager.MakeUpdateChannel()
	defer t.manager.CloseUpdateChannel(updateChannel)
	flushTimer := time.NewTimer(flushDelay)
	var numToFlush uint
	defer t.unregisterManagedExternalLeases()
	for {
		select {
		case update, ok := <-updateChannel:
			if !ok {
				return fmt.Errorf(
					"error sending update to: %s for: %s: receiver not keeping up with updates",
					conn.RemoteAddr(), conn.Username())
			}
			if err := conn.Encode(update); err != nil {
				return fmt.Errorf("error sending update: %s", err)
			}
			numToFlush++
			if !flushTimer.Stop() {
				select {
				case <-flushTimer.C:
				default:
				}
			}
			if len(updateChannel) < 1 && len(responseChannel) < 1 {
				flushTimer.Reset(flushDelay)
			}
		case update, ok := <-responseChannel:
			if !ok {
				return fmt.Errorf(
					"error sending response to: %s for: %s: receiver not keeping up with responses",
					conn.RemoteAddr(), conn.Username())
			}
			if err := conn.Encode(update); err != nil {
				return fmt.Errorf("error sending response: %s", err)
			}
			numToFlush++
			if !flushTimer.Stop() {
				select {
				case <-flushTimer.C:
				default:
				}
			}
			if len(updateChannel) < 1 && len(responseChannel) < 1 {
				flushTimer.Reset(flushDelay)
			}
		case <-flushTimer.C:
			if numToFlush > 1 {
				t.logger.Debugf(0, "flushing %d events\n", numToFlush)
			}
			numToFlush = 0
			if err := conn.Flush(); err != nil {
				return fmt.Errorf("error flushing update(s): %s", err)
			}
			heartbeatTimer.Reset(heartbeatDelay)
		case <-heartbeatTimer.C:
			err := conn.Encode(proto.Update{
				HealthStatus: t.manager.GetHealthStatus()})
			if err != nil {
				return fmt.Errorf("error writing heartbeat: %s", err)
			}
			numToFlush = 0
			if err := conn.Flush(); err != nil {
				return fmt.Errorf("error flushing heartbeat: %s", err)
			}
			heartbeatTimer.Reset(heartbeatDelay)
		case err := <-closeChannel:
			if err == nil {
				t.logger.Debugf(0, "update client disconnected: %s\n",
					conn.RemoteAddr())
				return nil
			}
			return err
		}
	}
}

func (t *srpcType) getUpdatesReader(decoder srpc.Decoder,
	heartbeatTimer *time.Timer) (
	<-chan error, <-chan proto.Update) {
	closeChannel := make(chan error)
	responseChannel := make(chan proto.Update, 16)
	go func() {
		for {
			var request proto.GetUpdatesRequest
			if err := decoder.Decode(&request); err != nil {
				if err == io.EOF {
					err = nil
				}
				closeChannel <- err
				return
			}
			heartbeatTimer.Reset(heartbeatDelay)
			if req := request.RegisterExternalLeasesRequest; req != nil {
				go t.registerManagedExternalLeases(*req)
			}
			update := proto.Update{HealthStatus: t.manager.GetHealthStatus()}
			select {
			case responseChannel <- update:
			default:
				close(responseChannel)
				return
			}
		}
	}()
	return closeChannel, responseChannel
}

// gRPC handler - streaming watch pattern
func (s *grpcServer) GetUpdates(req *pb.GetUpdatesRequest,
	stream pb.Hypervisor_GetUpdatesServer) error {
	// Create update channel from manager
	updateChannel := s.manager.MakeUpdateChannel()
	defer s.manager.CloseUpdateChannel(updateChannel)

	// Stream updates until context is cancelled
	for {
		select {
		case update, ok := <-updateChannel:
			if !ok {
				return fmt.Errorf("update channel closed")
			}

			// Convert SRPC Update to gRPC Update
			// The gRPC Update is simpler - just contains IP addresses of changed VMs
			pbUpdate := &pb.Update{}
			if update.HaveVMs {
				for ipStr := range update.VMs {
					ipAddr := net.ParseIP(ipStr)
					if shrunkIP := ipAddr.To4(); shrunkIP != nil {
						ipAddr = shrunkIP
					}
					pbUpdate.IpAddresses = append(pbUpdate.IpAddresses, []byte(ipAddr))
				}
			}

			// Send update to client
			if err := stream.Send(pbUpdate); err != nil {
				return fmt.Errorf("error sending update: %w", err)
			}

		case <-stream.Context().Done():
			// Client cancelled the stream
			s.logger.Debugf(0, "update client disconnected\n")
			return nil
		}
	}
}
