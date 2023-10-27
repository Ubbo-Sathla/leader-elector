package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ubbo-Sathla/leader-elector/pkg/etcd"
	"github.com/Ubbo-Sathla/leader-elector/pkg/vip"

	"github.com/alexflint/go-arg"
	"github.com/vishvananda/netlink"
	"google.golang.org/grpc/connectivity"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	Address   string
	NicSlave  string
	NicMaster string
	LeaseName string

	LeaseDuration int
}

var args struct {
	etcd.Etcd
	Config
}

func init() {

	arg.MustParse(&args)
	log.SetLevel(log.DebugLevel)
}

func main() {
	log.Debugf("args: %#v\n", args)

	StartCluster(args.Address, args.NicSlave, args.NicMaster, args.LeaseName, args.LeaseDuration)
}

func StartCluster(address string, nicSlave string, nicMaster string, leaseName string, leaseDuration int) {
	client, err := etcd.NewClient(&etcd.Etcd{
		CAFile:         args.CAFile,
		ClientCertFile: args.ClientCertFile,
		ClientKeyFile:  args.ClientKeyFile,
		Endpoints:      args.Endpoints,
	})
	if err != nil {
		panic(err)
	}

	id, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// listen for interrupts or the Linux SIGTERM signal and cancel
	// our context, which the leader election code will observe and
	// step down
	signalChan := make(chan os.Signal, 1)
	// Add Notification for Userland interrupt
	signal.Notify(signalChan, syscall.SIGINT)

	// Add Notification for SIGTERM (sent from Kubernetes)
	signal.Notify(signalChan, syscall.SIGTERM)

	go func() {
		<-signalChan
		log.Info("Received termination, signaling cluster shutdown")
		// Cancel the context, which will in turn cancel the leadership
		cancel()
		// Cancel the arp context, which will in turn stop any broadcasts
	}()

	//err = cluster.Network.DeleteIP()
	n, err := vip.NewConfig(address, nicSlave, nicMaster)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			log.Debug("Waiting for change: ", n.Interface())
			if n.Interface() == args.NicMaster {
				break
			}
			link, err := netlink.LinkByName(n.Interface())
			if err != nil {
				log.Fatal(err)
			}
			if link.Attrs().MasterIndex != 0 {
				signalChan <- syscall.SIGKILL
			}
			time.Sleep(time.Duration(leaseDuration) * time.Second)
		}
		log.Debug("End waiting for change: ", n.Interface())
	}()
	go func() {
		for {
			log.Debug(client.ActiveConnection().GetState())
			switch client.ActiveConnection().GetState() {
			case connectivity.TransientFailure:
				signalChan <- syscall.SIGKILL
			}
			time.Sleep(time.Duration(leaseDuration) * time.Second)
		}
	}()

	run := &runConfig{
		LeaseName:     fmt.Sprintf("/%s", leaseName),
		leaseID:       id,
		LeaseDuration: leaseDuration,
		onStartedLeading: func(ctx context.Context) {
			// As we're leading lets start the vip service

			log.Info("Starting the VIP service on the leader ", n.Interface())
			n.AddIP()
		},
		onStoppedLeading: func() {
			// we can do cleanup here
			log.Info("This node is becoming a follower within the cluster")
			n.DeleteIP()
			log.Fatal("lost leadership, restarting kube-vip")

		},
		onNewLeader: func(identity string) {
			// we're notified when new leader elected
			log.Infof("Node [%s] is assuming leadership of the cluster", identity)
		},
	}

	etcd.RunElectionOrDie(ctx, &etcd.LeaderElectionConfig{
		EtcdConfig:           etcd.ClientConfig{Client: client},
		Name:                 run.LeaseName,
		MemberID:             run.leaseID,
		LeaseDurationSeconds: int64(run.LeaseDuration),
		Callbacks: etcd.LeaderCallbacks{
			OnStartedLeading: run.onStartedLeading,
			OnStoppedLeading: run.onStoppedLeading,
			OnNewLeader:      run.onNewLeader,
		},
	})
}

type runConfig struct {
	LeaseName     string
	LeaseDuration int

	leaseID string

	// onStartedLeading is called when this member starts leading.
	onStartedLeading func(context.Context)
	// onStoppedLeading is called when this member stops leading.
	onStoppedLeading func()
	// onNewLeader is called when the client observes a leader that is
	// not the previously observed leader. This includes the first observed
	// leader when the client starts.
	onNewLeader func(identity string)
}
