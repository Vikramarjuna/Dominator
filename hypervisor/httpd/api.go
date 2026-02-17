package httpd

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/Cloud-Foundations/Dominator/hypervisor/manager"
	"github.com/Cloud-Foundations/Dominator/lib/html"
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

var htmlWriters []HtmlWriter

type state struct {
	manager *manager.Manager
}

func StartServer(portNum uint, managerObj *manager.Manager, daemon bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", portNum))
	if err != nil {
		return err
	}
	return ServeOnListener(listener, managerObj, daemon)
}

// ServeOnListener serves HTTP on an existing listener.
// Use this for cmux shared-port mode.
func ServeOnListener(listener net.Listener, managerObj *manager.Manager,
	daemon bool) error {
	myState := state{managerObj}
	html.HandleFunc("/", myState.statusHandler)
	html.HandleFunc("/listAvailableAddresses",
		myState.listAvailableAddressesHandler)
	html.HandleFunc("/listVolumeDirectories",
		myState.listVolumeDirectoriesHandler)
	html.HandleFunc("/listRegisteredAddresses",
		myState.listRegisteredAddressesHandler)
	html.HandleFunc("/listSubnets", myState.listSubnetsHandler)
	html.HandleFunc("/listVMs", myState.listVMsHandler)
	html.HandleFunc("/showVmBootLog", myState.showBootLogHandler)
	html.HandleFunc("/showVmLastPatchLog", myState.showLastPatchLogHandler)
	html.HandleFunc("/showVM", myState.showVMHandler)
	html.HandleFunc("/showVolumeDirectories",
		myState.showVolumeDirectoriesHandler)
	if daemon {
		go http.Serve(listener, nil)
	} else {
		http.Serve(listener, nil)
	}
	return nil
}

// NewHandler returns an http.Handler for SRPC and HTML pages.
// Use this for combined server mode where routing is done at the application level.
func NewHandler(managerObj *manager.Manager) http.Handler {
	myState := state{managerObj}
	mux := http.NewServeMux()
	html.ServeMuxHandleFunc(mux, "/", myState.statusHandler)
	html.ServeMuxHandleFunc(mux, "/listAvailableAddresses",
		myState.listAvailableAddressesHandler)
	html.ServeMuxHandleFunc(mux, "/listVolumeDirectories",
		myState.listVolumeDirectoriesHandler)
	html.ServeMuxHandleFunc(mux, "/listRegisteredAddresses",
		myState.listRegisteredAddressesHandler)
	html.ServeMuxHandleFunc(mux, "/listSubnets", myState.listSubnetsHandler)
	html.ServeMuxHandleFunc(mux, "/listVMs", myState.listVMsHandler)
	html.ServeMuxHandleFunc(mux, "/showVmBootLog", myState.showBootLogHandler)
	html.ServeMuxHandleFunc(mux, "/showVmLastPatchLog", myState.showLastPatchLogHandler)
	html.ServeMuxHandleFunc(mux, "/showVM", myState.showVMHandler)
	html.ServeMuxHandleFunc(mux, "/showVolumeDirectories",
		myState.showVolumeDirectoriesHandler)
	return mux
}

func AddHtmlWriter(htmlWriter HtmlWriter) {
	htmlWriters = append(htmlWriters, htmlWriter)
}
