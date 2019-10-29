package main

import (
	"context"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/protolambda/go-libp2p-model-net/mocknet"
	"log"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	env := mocknet.NewModelEnvironment()
	mn := mocknet.NewMocknet(ctx, env)
	n := 20
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
	stepDuration := 500 * time.Millisecond
	timeSinceHeartBeats := time.Duration(0)
	heartbeatInterval := pubsub.GossipSubHeartbeatInterval
	for i := 0; i < 100; i++ {
		// drain all events
		// TODO: all at same time?
		log.Printf("round %d, time: %s", i, env.Now().String())
		for id, ch := range chosts {
			workLoop: for {
				select {
					case <-ch.workLeft:
						log.Printf("performed work for %s", id.ShortString())
				default:
					log.Printf("completed work for %s", id.ShortString())
					break workLoop
				}
			}
		}
		log.Printf("running %s of network traffic", stepDuration.String())
		env.StepDelta(stepDuration)
		timeSinceHeartBeats += stepDuration
		if timeSinceHeartBeats >= heartbeatInterval {
			timeSinceHeartBeats -= heartbeatInterval
			for id, ch := range chosts {
				log.Printf("performing gs heartbeat of %s", id.ShortString())
				ch.heartbeat()
			}
		}
	}
	cancel()
}

type ControlledHost struct {
	h host.Host
	workLeft <-chan struct{}
	heartbeat func()
	psCtx context.Context
	psEval chan<- func()
	ps *pubsub.PubSub
}

func initHost(ctx context.Context, env mocknet.Environment, h host.Host) (*ControlledHost, error) {
	ch := &ControlledHost{h: h}
	var wg sync.WaitGroup
	wg.Add(2)
	hbinit := func(ctx context.Context, eval chan<- func(), heartbeat func()) {
		ch.psCtx = ctx
		ch.psEval = eval
		ch.heartbeat = heartbeat
		wg.Done()
	}
	evControl := func(workLeft <-chan struct{}) {
		ch.workLeft = workLeft
		wg.Done()
	}
	ps, err := pubsub.NewCustomGossipSub(ctx, h, env, hbinit, pubsub.WithEventControl(evControl))
	if err != nil {
		return nil, err
	}
	ch.ps = ps
	wg.Wait()
	return ch, nil
}
