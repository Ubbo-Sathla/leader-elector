package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/Ubbo-Sathla/leader-elector/pkg/election"
	"github.com/alexflint/go-arg"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

var (
	args struct {
		LockName      string        `arg:"--election,env:ELECTION_NAME" default:"default" help:"Name of this election"`
		Namespace     string        `arg:"env:ELECTION_NAMESPACE" default:"default" help:"Namespace of this election"`
		LockType      string        `arg:"env:ELECTION_TYPE" default:"configmaps" help:"Resource lock type, must be one of the following: configmaps, endpoints, leases"`
		RenewDeadline time.Duration `arg:"--renew-deadline,env:ELECTION_RENEW_DEADLINE" default:"10s" help:"Duration that the acting leader will retry refreshing leadership before giving up"`
		RetryPeriod   time.Duration `arg:"--retry-period,env:ELECTION_RETRY_PERIOD" default:"2s" help:"Duration between each action retry"`
		LeaseDuration time.Duration `arg:"--lease-duration,env:ELECTION_LEASE_DURATION" default:"15s" help:"Duration that non-leader candidates will wait after observing a leadership renewal until attempting to acquire leadership of a led but unrenewed leader slot"`
		Port          string        `arg:"env:ELECTION_PORT" default:"4040" help:"Port on which to query the leader"`
	}
	leader Leader
)

// Leader contains the name of the current leader of this election
type Leader struct {
	Name string `json:"name"`
}

func leaderHandler(res http.ResponseWriter, req *http.Request) {
	data, err := json.Marshal(leader)
	if err != nil {
		klog.Errorf("Error while marshaling leader response: %s", err.Error())
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
	res.Write(data)
}

func main() {
	arg.MustParse(&args)

	vip, _ := netlink.ParseAddr(os.Getenv("address"))

	// configuring context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// configuring signal handling
	terminationSignal := make(chan os.Signal, 1)
	signal.Notify(terminationSignal, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-terminationSignal
		klog.Infoln("Received termination signal, shutting down")
		cancel()
	}()

	// configuring HTTP server
	http.HandleFunc("/", leaderHandler)
	server := &http.Server{Addr: ":" + args.Port, Handler: nil}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			klog.Fatal(err)
		}
	}()
	go func() {
		for {
			if leader.Name == os.Getenv("HOSTNAME") {
				links, err := netlink.LinkList()

				if err != nil {
					klog.Fatal(err)
				}
				for _, link := range links {

					reg := regexp.MustCompile(`^eth.+|^ens.+|^bond.+|^br0`)
					if !reg.MatchString(link.Attrs().Name) {
						//klog.Infoln("not match: ", link.Attrs().Name)
						continue
					}
					address, err := netlink.AddrList(link, netlink.FAMILY_V4)
					if err != nil {
						klog.Errorf("get %s ip address err: %s", link.Attrs().Name, err)
					}
					klog.Infoln(link.Attrs().Name, address)

					// 判断网卡是否正确 true 为 需要配置IP的网卡
					check := false
					// 判断IP是否已经配置 true 为已经配置
					configTag := false
					for _, addr := range address {
						ipStr := os.Getenv("IP")
						ip := net.ParseIP(ipStr)
						if addr.IP.Equal(ip) {
							check = true
						}
						if addr.IP.Equal(vip.IP) {
							configTag = true
						}
					}

					if check {
						// 如果没有配置IP 则需要配置
						if !configTag {
							err = netlink.AddrAdd(link, vip)
							if err != nil {
								klog.Errorf("config ip err: ", err)
							} else {
								klog.Infof("config ip %s on %s", vip.String(), link.Attrs().Name)
							}
						}
					} else {
						// 如果配置了IP 则需要删除
						if configTag {
							err = netlink.AddrDel(link, vip)
							if err != nil {
								klog.Errorf("delete ip err: ", err)
							}
						}
					}
				}
			} else {
				links, err := netlink.LinkList()
				if err != nil {
					klog.Fatal(err)
				}
				for _, link := range links {
					address, err := netlink.AddrList(link, netlink.FAMILY_V4)
					for _, addr := range address {
						if addr.IP.Equal(vip.IP) {
							err = netlink.AddrDel(link, vip)
							if err != nil {
								klog.Errorf("delete ip err: ", err)
							}
						}
					}
				}
			}
			klog.Infoln("check ip")
			time.Sleep(5 * time.Second)
		}
	}()
	// configuring Leader Election loop
	callback := func(name string) {
		klog.Infof("Currently leading: %s", name)
		leader = Leader{name}
	}

	electionConfig := election.Config{
		LockName:      args.LockName,
		LockNamespace: args.Namespace,
		LockType:      args.LockType,
		RenewDeadline: args.RenewDeadline,
		RetryPeriod:   args.RetryPeriod,
		LeaseDuration: args.LeaseDuration,
		Callback:      callback,
	}
	election.Run(ctx, electionConfig)

	// gracefully stop HTTP server
	srvCtx, srvCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer srvCancel()
	if err := server.Shutdown(srvCtx); err != nil {
		klog.Fatal(err)
	}
}
