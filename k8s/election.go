package k8s

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// Leader election timings, matching the client-go recommendations for a
// controller that is not latency critical: losing leadership briefly only
// delays a status write, it never interrupts traffic.
const (
	leaseDuration = 30 * time.Second
	renewDeadline = 20 * time.Second
	retryPeriod   = 5 * time.Second
)

// leaderState - Whether this replica is the one writing status.
//
// Only status writes are gated on leadership. Every replica watches the
// cluster and serves traffic, so losing the lease never takes a pod out of the
// data path -- it just stops it from fighting its peers over
// `.status.loadBalancer`.
type leaderState struct {
	isLeader atomic.Bool
}

func newLeaderState() *leaderState {
	return &leaderState{}
}

func (l *leaderState) IsLeader() bool {
	return l != nil && l.isLeader.Load()
}

// runLeaderElection - Campaigns for the status-writing lease until the context
// is cancelled.
func (c *Controller) runLeaderElection(ctx context.Context) {
	log := logger.GetGlobal()

	identity := os.Getenv("POD_NAME")
	if identity == "" {
		hostname, err := os.Hostname()
		if err != nil {
			log.Errorf("Cannot determine a leader election identity: %s", err)
			return
		}

		identity = hostname
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      c.opts.ElectionID,
			Namespace: c.opts.Namespace,
		},
		Client: c.core.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: identity,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(context.Context) {
				log.Infof("Became leader (%s): now publishing status", identity)
				c.leader.isLeader.Store(true)

				// Status was not being written while another replica held the
				// lease, so refresh everything now.
				c.enqueue()
			},
			OnStoppedLeading: func() {
				log.Warnf("Lost leadership (%s): no longer publishing status", identity)
				c.leader.isLeader.Store(false)
			},
			OnNewLeader: func(newLeader string) {
				if newLeader != identity {
					log.Debugf("Leader is now %s", newLeader)
				}
			},
		},
	})
}
