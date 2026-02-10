package srpc

import "crypto/tls"

// GetAuthFromTLS extracts authentication information from a TLS connection state.
// This can be used by other protocols (like gRPC) that use the same TLS certificates
// and want to reuse SRPC's authentication model.
//
// The returned AuthInformation contains Username and GroupList extracted from
// the client certificate. The HaveMethodAccess field is not set and should be
// determined by the calling protocol's authorization logic.
func GetAuthFromTLS(state tls.ConnectionState) (*AuthInformation, error) {
	username, _, groupList, err := getAuth(state)
	if err != nil {
		return nil, err
	}
	return &AuthInformation{
		Username:  username,
		GroupList: groupList,
		// HaveMethodAccess is protocol-specific and must be set by caller
	}, nil
}

// GetPermittedMethodsFromTLS returns the permitted methods from a TLS connection state.
// This is useful for protocols that want to implement method-level authorization
// similar to SRPC.
func GetPermittedMethodsFromTLS(state tls.ConnectionState) (map[string]struct{}, error) {
	_, permittedMethods, _, err := getAuth(state)
	return permittedMethods, err
}

