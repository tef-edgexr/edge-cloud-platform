// Copyright 2022 MobiledgeX, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"strconv"
	"strings"
)

var maxTcpPorts int = 1000
var maxUdpPorts int = 10000
var maxEnvoyUdpPorts int = 1000
var MaxK8sUdpPorts int = 1000
var minUDPPktSize int64 = 1500
var maxUDPPktSize int64 = 50000

type PortSpec struct {
	Proto           string
	Port            string
	EndPort         string // mfw XXX ? why two type and parse rtns for AppPort? (3 actually kube.go is another)
	Tls             bool
	Nginx           bool
	MaxPktSize      int64
	InternalVisOnly bool
	ID              string
	PathPrefix      string
	ServiceName     string
}

func ParsePorts(accessPorts string) ([]PortSpec, error) {
	var baseport int64
	var endport int64
	var err error

	tcpPortCount := 0
	udpPortCount := 0

	ports := []PortSpec{}
	pstrs := strings.Split(accessPorts, ",")

	for _, pstr := range pstrs {
		pp := strings.Split(pstr, ":")
		if len(pp) < 2 {
			return nil, fmt.Errorf("invalid AccessPorts format '%s'", pstr)
		}
		annotations := make(map[string]string)
		for _, kv := range pp[2:] {
			if kv == "" {
				return nil, fmt.Errorf("invalid AccessPorts annotation %s for port %s, expected format is either key or key=val", kv, pp[1])
			}
			keyval := strings.SplitN(kv, "=", 2)
			if len(keyval) == 1 {
				// boolean annotation
				annotations[kv] = "true"
			} else if len(keyval) == 2 {
				annotations[keyval[0]] = keyval[1]
			} else {
				return nil, fmt.Errorf("invalid AccessPorts annotation %s for port %s, expected format is either key or key=val", kv, pp[1])
			}
		}
		// within each pp[1], we may have a hypenated range of ports ex: udp:M-N inclusive
		portrange := strings.Split(pp[1], "-")
		// len of portrange is 2 if a range 1 if simple port value
		// in either case, baseport is the first elem of portrange
		baseport, err = strconv.ParseInt(portrange[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("unable to convert port range base value")
		}
		if len(portrange) == 2 {
			endport, err = strconv.ParseInt(portrange[1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("unable to convert port range base value")
			}
		} else {
			endport = baseport
		}

		if (baseport < 1 || baseport > 65535) ||
			(endport < 1 || endport > 65535) {
			return nil, fmt.Errorf("App ports out of range")
		}
		if endport < baseport {
			// after some debate, error on this potential typo len(portrange)
			return nil, fmt.Errorf("App ports out of range")
		}

		if baseport == endport {
			// ex: tcp:5000-5000 or just portrange len = 1
			endport = 0
		}

		proto := strings.ToLower(pp[0])
		if proto != "tcp" && proto != "udp" && proto != "http" {
			return nil, fmt.Errorf("Unsupported protocol: %s", pp[0])
		}

		portCount := 1
		if endport != 0 {
			portCount = int(endport-baseport) + 1
		}
		if proto == "tcp" {
			tcpPortCount = tcpPortCount + portCount
		} else { // udp
			udpPortCount = udpPortCount + portCount
		}

		portSpec := PortSpec{
			Proto:   proto,
			Port:    strconv.FormatInt(baseport, 10),
			EndPort: strconv.FormatInt(endport, 10),
		}
		for key, val := range annotations {
			switch key {
			case "tls":
				if portSpec.Proto != "tcp" && portSpec.Proto != "http" {
					return nil, fmt.Errorf("Invalid protocol %s, not available for tls support", portSpec.Proto)
				}
				portSpec.Tls = true
			case "nginx":
				if portSpec.Proto != "udp" {
					return nil, fmt.Errorf("Invalid annotation \"nginx\" for %s ports", portSpec.Proto)
				}
				portSpec.Nginx = true
			case "maxpktsize":
				if portSpec.Proto != "udp" {
					return nil, fmt.Errorf("Invalid annotation \"maxpktsize\" for %s ports, only valid for UDP protocol", portSpec.Proto)
				}
				maxPktSize, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("unable to convert pkt size value: %s", val)
				}
				if maxPktSize < minUDPPktSize || maxPktSize > maxUDPPktSize {
					return nil, fmt.Errorf("Invalid maxpktsize, should be between range %v to %v (exclusive)", minUDPPktSize, maxUDPPktSize)
				}
				portSpec.MaxPktSize = maxPktSize
			case "intvis":
				portSpec.InternalVisOnly = true
			case "id":
				portSpec.ID = val
			case "pathprefix":
				if portSpec.Proto != "http" {
					return nil, fmt.Errorf("invalid annotation pathprefix on port %s, only allowed on http ports", portSpec.Port)
				}
				portSpec.PathPrefix = val
			case "svcname":
				portSpec.ServiceName = val
			default:
				return nil, fmt.Errorf("unrecognized annotation %s for port %s", key+"="+val, pp[1])
			}
		}
		ports = append(ports, portSpec)
	}
	if tcpPortCount > maxTcpPorts {
		return nil, fmt.Errorf("Not allowed to specify more than %d tcp ports", maxTcpPorts)
	}
	if udpPortCount > maxUdpPorts {
		return nil, fmt.Errorf("Not allowed to specify more than %d udp ports", maxUdpPorts)
	}
	if udpPortCount > maxEnvoyUdpPorts {
		for i, _ := range ports {
			if ports[i].Proto == "udp" {
				ports[i].Nginx = true
			}
		}
	}

	return ports, nil
}
