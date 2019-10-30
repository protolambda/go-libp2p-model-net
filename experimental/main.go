package main

import (
	"context"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/protolambda/go-libp2p-model-net/mocknet"
	"log"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	env := mocknet.NewModelEnvironment()
	mn := mocknet.NewMocknet(ctx, env)
	n := 2
	for i := 0; i < n; i++ {
		if _, err := mn.GenPeer(); err != nil {
			log.Fatalln(err)
		}
	}
	mn.SetLinkDefaults(mocknet.LinkOptions{
		Latency:   2 * time.Millisecond,
		Bandwidth: 0,
	})
	if err := mn.LinkAll(); err != nil {
		log.Fatalln(err)
	}
	if err := mn.ConnectAllButSelf(); err != nil {
		log.Fatalln(err)
	}
	chosts := make(map[peer.ID]*ControlledHost)
	for _, h := range mn.Hosts() {
		ch, err := initHost(ctx, env, h)
		if err != nil {
			log.Fatalln(err)
		}
		chosts[h.ID()] = ch
	}
	for _, ch := range chosts {
		ch.ps.Start(false)
	}
	processEvents := func() {
		for id, ch := range chosts {
			if ch.killed {
				continue
			}
			log.Printf("performing work for %s", id.ShortString())
			for {
				free, exit := ch.ps.ProcessNextEvent()
				if exit {
					// cleanup if we are exiting
					ch.ps.Cleanup()
					ch.killed = true
					log.Printf("completed all work for %s", id.ShortString())
				}
				// if there is no work to do, or if the node is killed, then stop work.
				if free || exit {
					break
				}
			}
		}
	}
	stepDuration := 500 * time.Millisecond
	timeSinceHeartBeats := time.Duration(0)
	heartbeatInterval := pubsub.GossipSubHeartbeatInterval
	for i := 0; i < 100; i++ {
		// drain all events
		// TODO: all at same time?
		log.Printf("round %d, time: %s", i, env.Now().String())

		// Network traffic
		log.Printf("running %s of network traffic", stepDuration.String())
		env.StepDelta(stepDuration)

		// Process messages
		processEvents()

		// run heartbeats if it's time to do so
		timeSinceHeartBeats += stepDuration
		if timeSinceHeartBeats >= heartbeatInterval {
			timeSinceHeartBeats -= heartbeatInterval
			for id, ch := range chosts {
				log.Printf("performing gs heartbeat of %s", id.ShortString())
				ch.gs.Heartbeat()
			}
		}
	}
	cancel()
}

type ControlledHost struct {
	h host.Host
	gs *pubsub.GossipSubRouter
	ps *pubsub.PubSub
	killed bool
}

func initHost(ctx context.Context, env mocknet.Environment, h host.Host) (*ControlledHost, error) {
	ch := &ControlledHost{h: h}
	ch.gs = pubsub.NewGossipSub()
	ps, err := pubsub.NewPubSub(ctx, h, ch.gs, pubsub.WithClock(env))
	if err != nil {
		return nil, err
	}
	ch.ps = ps
	return ch, nil
}
