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

package proxy

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"strconv"
	"strings"

	dme "github.com/edgexr/edge-cloud-platform/api/distributed_match_engine"
	"github.com/edgexr/edge-cloud-platform/api/edgeproto"
	"github.com/edgexr/edge-cloud-platform/pkg/access"
	"github.com/edgexr/edge-cloud-platform/pkg/cloudcommon"
	"github.com/edgexr/edge-cloud-platform/pkg/dockermgmt"
	"github.com/edgexr/edge-cloud-platform/pkg/log"
	"github.com/edgexr/edge-cloud-platform/pkg/platform/pc"
	ssh "github.com/edgexr/golang-ssh"
)

// Nginx is used to proxy connections from the external network to
// the internal cloudlet clusters. It is not doing any load-balancing.
// Access to an AppInst is handled by a per-AppInst dedicated
// nginx instance for better isolation.

var NginxL7Name = "nginxL7"

var nginxConfT *template.Template

// defaultConcurrentConnsPerIP is the default DOS protection setting for connections per source IP
const defaultConcurrentConnsPerIP uint64 = 100
const defaultWorkerConns int = 1024

// TCP is in envoy, which does not have concurrent connections per IP, but rather
// just concurrent connections overall
func getTCPConcurrentConnections() (uint64, error) {
	var err error
	connStr := os.Getenv("MEX_LB_CONCURRENT_TCP_CONNS")
	conns := defaultConcurrentConns
	if connStr != "" {
		conns, err = strconv.ParseUint(connStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return conns, nil
}

func getUDPConcurrentConnections() (uint64, error) {
	var err error
	connStr := os.Getenv("MEX_LB_CONCURRENT_UDP_CONNS")
	conns := defaultConcurrentConnsPerIP
	if connStr != "" {
		conns, err = strconv.ParseUint(connStr, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return conns, nil
}

func init() {
	nginxConfT = template.Must(template.New("conf").Parse(nginxConf))
}

func CheckProtocols(name string, ports []edgeproto.InstPort) (bool, bool) {
	needEnvoy := false
	needNginx := false
	for _, p := range ports {
		switch p.Proto {
		case dme.LProto_L_PROTO_HTTP:
			fallthrough
		case dme.LProto_L_PROTO_TCP:
			needEnvoy = true
		case dme.LProto_L_PROTO_UDP:
			if p.Nginx {
				needNginx = true
			} else {
				needEnvoy = true
			}
		}
	}
	return needEnvoy, needNginx
}

func getNginxContainerName(name string) string {
	return "nginx" + name
}

type ProxyConfig struct {
	ListenIP    string
	DestIP      string
	ListenIPV6  string
	DestIPV6    string
	SkipHCPorts string
}

func CreateNginxProxy(ctx context.Context, client ssh.Client, name, envoyImage, nginxImage string, config *ProxyConfig, appInst *edgeproto.AppInst, authAPI cloudcommon.RegistryAuthApi, ops ...Op) error {

	log.SpanLog(ctx, log.DebugLevelInfra, "CreateNginxProxy", "name", name, "config", config)
	containerName := getNginxContainerName(name)

	// check to see whether nginx or envoy is needed (or both)
	envoyNeeded, nginxNeeded := CheckProtocols(name, appInst.MappedPorts)
	if envoyNeeded {
		err := CreateEnvoyProxy(ctx, client, name, envoyImage, config, appInst, authAPI, ops...)
		if err != nil {
			log.SpanLog(ctx, log.DebugLevelInfra, "CreateEnvoyProxy failed ", "err", err)
			return fmt.Errorf("Create Envoy Proxy failed, %v", err)
		}
	}
	if !nginxNeeded {
		return nil
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "create nginx", "name", name, "config", config, "ports", appInst.MappedPorts)
	opts := Options{}
	opts.Apply(ops)

	// if nginx image is not present, ensure pull credentials are present if needed
	present, err := dockermgmt.DockerImagePresent(ctx, client, nginxImage)
	if err != nil || !present {
		err = dockermgmt.SeedDockerSecret(ctx, client, nginxImage, authAPI)
		if err != nil {
			return err
		}
	}

	out, err := client.Output("pwd")
	if err != nil {
		return err
	}
	pwd := strings.TrimSpace(string(out))

	dir := pwd + "/nginx/" + name
	log.SpanLog(ctx, log.DebugLevelInfra, "nginx remote dir", "name", name, "dir", dir)

	err = pc.Run(client, "mkdir -p "+dir)
	if err != nil {
		return err
	}

	usesTLS := false
	err = pc.Run(client, "ls cert.pem && ls key.pem")
	if err == nil {
		usesTLS = true
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "nginx certs check",
		"name", name, "usesTLS", usesTLS)

	errlogFile := dir + "/err.log"
	err = pc.Run(client, "touch "+errlogFile)
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra,
			"nginx %s can't create file %s", name, errlogFile)
		return err
	}
	accesslogFile := dir + "/access.log"
	err = pc.Run(client, "touch "+accesslogFile)
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra,
			"nginx %s can't create file %s", name, accesslogFile)
		return err
	}
	nconfName := dir + "/nginx.conf"
	err = createNginxConf(ctx, client, nconfName, name, config, appInst, usesTLS)
	if err != nil {
		return fmt.Errorf("create nginx.conf failed, %v", err)
	}

	cmdArgs := []string{"run", "-d", "-l edge-cloud", "--restart=unless-stopped", "--name", containerName}
	if opts.DockerPublishPorts {
		cmdArgs = append(cmdArgs, dockermgmt.GetDockerPortString(appInst.MappedPorts, dockermgmt.UsePublicPortInContainer, dockermgmt.NginxProxy, config.ListenIP, config.ListenIPV6)...)
	}
	if opts.DockerNetwork != "" {
		// For dind, we use the network which the dind cluster is on.
		cmdArgs = append(cmdArgs, "--network", opts.DockerNetwork)
	}
	if usesTLS {
		cmdArgs = append(cmdArgs, "-v", pwd+"/cert.pem:/etc/ssl/certs/server.crt")
		cmdArgs = append(cmdArgs, "-v", pwd+"/key.pem:/etc/ssl/certs/server.key")
	}
	cmdArgs = append(cmdArgs,
		"-v", dir+":/var/www/.cache",
		"-v", "/etc/ssl/certs:/etc/ssl/certs",
		"-v", errlogFile+":/var/log/nginx/error.log",
		"-v", accesslogFile+":/var/log/nginx/access.log",
		"-v", nconfName+":/etc/nginx/nginx.conf",
		nginxImage)
	cmd := "docker " + strings.Join(cmdArgs, " ")
	log.SpanLog(ctx, log.DebugLevelInfra, "nginx docker command", "containerName", containerName,
		"cmd", cmd)
	out, err = client.Output(cmd)
	if err != nil {
		return fmt.Errorf("can't create nginx container %s, %s, %v", name, out, err)
	}
	log.SpanLog(ctx, log.DebugLevelInfra, "created nginx container", "containerName", containerName)
	return nil
}

func createNginxConf(ctx context.Context, client ssh.Client, confname, name string, config *ProxyConfig, appInst *edgeproto.AppInst, usesTLS bool) error {
	spec := ProxySpec{
		Name:        name,
		UsesTLS:     usesTLS,
		MetricIP:    config.ListenIP,
		MetricPort:  cloudcommon.ProxyMetricsPort,
		WorkerConns: defaultWorkerConns,
	}
	if spec.MetricIP == "" {
		spec.MetricIP = config.ListenIPV6
	}

	portCount := 0

	udpconns, err := getUDPConcurrentConnections()
	if err != nil {
		return err
	}
	proxyIPPairs := []struct {
		listenIP string
		destIP   string
	}{
		{config.ListenIP, config.DestIP},
		{config.ListenIPV6, config.DestIPV6},
	}
	for _, proxyIPPair := range proxyIPPairs {
		if proxyIPPair.destIP == "" {
			continue
		}
		for _, p := range appInst.MappedPorts {
			serviceBackendIP, err := getBackendIpToUse(ctx, appInst, &p, proxyIPPair.destIP)
			if err != nil {
				return err
			}
			if p.Proto == dme.LProto_L_PROTO_UDP {
				if !p.Nginx { // use envoy
					continue
				}
				udpPort := UDPSpecDetail{
					ListenIP:        proxyIPPair.listenIP,
					BackendIP:       serviceBackendIP,
					BackendPort:     p.InternalPort,
					ConcurrentConns: udpconns,
				}
				endPort := p.EndPort
				if endPort == 0 {
					endPort = p.PublicPort
					portCount = portCount + 1
				} else {
					portCount = int((p.EndPort-p.InternalPort)+1) + portCount
					// if we have a port range, the internal ports and external ports must match
					if p.InternalPort != p.PublicPort {
						return fmt.Errorf("public and internal ports must match when port range in use")
					}
				}
				for pnum := p.PublicPort; pnum <= endPort; pnum++ {
					udpPort.NginxListenPorts = append(udpPort.NginxListenPorts, pnum)
				}
				// if there is more than one listen port, we don't use the backend port as the
				// listen port is used as the backend port in the case of a range
				if len(udpPort.NginxListenPorts) > 1 {
					udpPort.BackendPort = 0
				}
				spec.UDPSpec = append(spec.UDPSpec, &udpPort)
			}
		}
	}
	// need to have more worker connections than ports otherwise nginx will crash
	if portCount > 1000 {
		spec.WorkerConns = int(float64(portCount) * 1.2)
	}

	log.SpanLog(ctx, log.DebugLevelInfra, "create nginx conf", "name", name)
	buf := bytes.Buffer{}
	err = nginxConfT.Execute(&buf, &spec)
	if err != nil {
		return err
	}
	err = pc.WriteFile(client, confname, buf.String(), "nginx.conf", pc.NoSudo)
	if err != nil {
		log.SpanLog(ctx, log.DebugLevelInfra, "write nginx.conf failed",
			"name", name, "err", err)
		return err
	}
	return nil
}

type ProxySpec struct {
	Name        string
	UDPSpec     []*UDPSpecDetail
	TCPSpec     []*TCPSpecDetail
	UsesTLS     bool // To be removed
	MetricIP    string
	MetricPort  int32
	MetricUDS   bool
	CertName    string
	WorkerConns int
}

type TCPSpecDetail struct {
	ListenIP        string
	ListenPort      int32
	BackendIP       string
	BackendPort     int32
	ConcurrentConns uint64
	UseTLS          bool // for port specific TLS termination
	HealthCheck     bool
	IPTag           string
}

type UDPSpecDetail struct {
	ListenIP         string
	ListenPort       int32
	NginxListenPorts []int32
	BackendIP        string
	BackendPort      int32
	ConcurrentConns  uint64
	MaxPktSize       int64
	IPTag            string
}

var nginxConf = `
user  nginx;
worker_processes  1;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events {
    worker_connections  {{.WorkerConns}};
}

stream { 
    limit_conn_zone $binary_remote_addr zone=ipaddr:10m;
	{{- range .UDPSpec}}
	server {
		limit_conn ipaddr {{.ConcurrentConns}}; 
		{{range $portnum := .NginxListenPorts}}
		listen {{$portnum}} udp; 
		{{end}}
		{{if eq .BackendPort 0}}
		proxy_pass {{.BackendIP}}:$server_port;
		{{- end}}
		{{if ne .BackendPort 0}}
		proxy_pass {{.BackendIP}}:{{.BackendPort}};
		{{- end}}
	}
	{{- end}}
}
`

func DeleteNginxProxy(ctx context.Context, client ssh.Client, name string) error {
	log.SpanLog(ctx, log.DebugLevelInfra, "delete nginx", "name", name)
	containerName := getNginxContainerName(name)
	out, err := client.Output("docker kill " + containerName)
	log.SpanLog(ctx, log.DebugLevelInfra, "kill nginx result", "out", out, "err", err)

	nginxDir := "nginx/" + name
	out, err = client.Output("rm -rf " + nginxDir)
	log.SpanLog(ctx, log.DebugLevelInfra, "delete nginx dir result", "name", name, "dir", nginxDir, "out", out, "err", err)

	out, err = client.Output("docker rm -f " + containerName)
	log.SpanLog(ctx, log.DebugLevelInfra, "rm nginx result", "out", out, "err", err)
	if err != nil && !strings.Contains(string(out), "No such container") {
		// delete the envoy proxy for best effort
		DeleteEnvoyProxy(ctx, client, name)
		return fmt.Errorf("can't remove nginx container %s, %s, %v", name, out, err)
	}

	log.SpanLog(ctx, log.DebugLevelInfra, "deleted nginx", "containerName", containerName)
	return DeleteEnvoyProxy(ctx, client, name)
}

type Options struct {
	DockerPublishPorts bool
	DockerNetwork      string
	Cert               *access.TLSCert
	DockerUser         string
	MetricIP           string
	MetricUDS          bool // Unix Domain Socket
}

type Op func(opts *Options)

func WithDockerNetwork(network string) Op {
	return func(opts *Options) {
		opts.DockerNetwork = network
	}
}

func WithDockerPublishPorts() Op {
	return func(opts *Options) {
		opts.DockerPublishPorts = true
	}
}

func WithTLSCert(cert *access.TLSCert) Op {
	return func(opts *Options) {
		opts.Cert = cert
		opts.Cert.CommonName = strings.Replace(opts.Cert.CommonName, "*", "_", 1)
	}
}

func WithDockerUser(user string) Op {
	return func(opts *Options) {
		opts.DockerUser = user
	}
}

func WithMetricEndpoint(endpoint string) Op {
	return func(opts *Options) {
		if endpoint == cloudcommon.ProxyMetricsListenUDS {
			opts.MetricUDS = true
		} else {
			opts.MetricIP = endpoint
		}
	}
}

func (o *Options) Apply(ops []Op) {
	for _, op := range ops {
		op(o)
	}
}
