package etcd

import (
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

type Etcd struct {
	CAFile         string
	ClientCertFile string
	ClientKeyFile  string
	Endpoints      []string
}

func NewClient(e *Etcd) (*clientv3.Client, error) {
	tlsInfo := transport.TLSInfo{
		TrustedCAFile: e.CAFile,
		CertFile:      e.ClientCertFile,
		KeyFile:       e.ClientKeyFile,
	}

	clientTLS, err := tlsInfo.ClientConfig()
	if err != nil {
		return nil, err
	}

	return clientv3.New(clientv3.Config{
		Endpoints:            e.Endpoints,
		TLS:                  clientTLS,
		DialTimeout:          5 * time.Second,
		DialKeepAliveTimeout: 10 * time.Second,
	})

}
