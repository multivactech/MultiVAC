// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package socks

import "fmt"

// ProxiedAddr is used for proxy address.
type ProxiedAddr struct {
	Net  string
	Host string
	Port int
}

// Network returns the net of ProxiedAddr.
func (a *ProxiedAddr) Network() string {
	return a.Net
}

// String will print the ip address and port of proxied address.
func (a *ProxiedAddr) String() string {
	return fmt.Sprintf("%s:%d", a.Host, a.Port)
}
