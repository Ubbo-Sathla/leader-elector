package main

import (
	"context"
	"fmt"
	"github.com/Ubbo-Sathla/leader-elector/pkg/etcd"
	"github.com/alexflint/go-arg"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/connectivity"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var args struct {
	etcd.Etcd
}

func init() {
	arg.MustParse(&args)
	log.SetLevel(log.DebugLevel)
}

func main() {
	fmt.Println(args)
	StartCluster()
}

func StartCluster() {
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

	go func() {
		for {
			log.Debug(client.ActiveConnection().GetState())
			switch client.ActiveConnection().GetState() {
			case connectivity.TransientFailure:
				signalChan <- syscall.SIGKILL
			}
			time.Sleep(5 * time.Second)

		}

	}()

	run := &runConfig{
		LeaseName:     "/kubevip",
		leaseID:       id,
		LeaseDuration: 5,
		onStartedLeading: func(ctx context.Context) {
			// As we're leading lets start the vip service

			log.Print("Starting the VIP service on the leader")

		},
		onStoppedLeading: func() {
			// we can do cleanup here
			log.Info("This node is becoming a follower within the cluster")

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
