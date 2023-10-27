# k8s vip

> use etcd to elector master which config vip , not use keepalived vrrp. code copy from https://github.com/kube-vip/kube-vip

## system service

```bash
cat > /usr/lib/systemd/system/kubevip.service << EOF
[Unit]
Description=kubevip: The Kubernetes Vip
Documentation=https://kubernetes.io/docs/

[Service]
ExecStart=/usr/local/bin/kubevip --cafile /etc/kubernetes/pki/etcd/ca.crt  --clientcertfile /etc/kubernetes/pki/etcd/peer.crt  --clientkeyfile /etc/kubernetes/pki/etcd/peer.key --endpoints https://10.119.1.19:2379 https://127.0.0.1:2379 --address 192.168.1.1/24 --nicslave bond4 --nicmaster br0 --leaseduration 5 --leasename cloudvip
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
```