package main

import (
	"flag"
	"os"
	"runtime/pprof"

	"github.com/AsynkronIT/gam/actor"
	"github.com/AsynkronIT/gam/remoting"

	"log"
	"sync"

	"runtime"
	"time"

	"github.com/AsynkronIT/gam/examples/remotebenchmark/messages"
)

type localActor struct {
	count        int
	wgStop       *sync.WaitGroup
	messageCount int
}

func (state *localActor) Receive(context actor.Context) {
	switch context.Message().(type) {
	case *messages.Pong:
		state.count++
		if state.count%50000 == 0 {
			log.Println(state.count)
		}
		if state.count == state.messageCount {
			state.wgStop.Done()
		}
	}
}

func newLocalActor(stop *sync.WaitGroup, messageCount int) actor.Producer {
	return func() actor.Actor {
		return &localActor{
			wgStop:       stop,
			messageCount: messageCount,
		}
	}
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.GOMAXPROCS(runtime.NumCPU() * 1)
	runtime.GC()

	var wg sync.WaitGroup

	messageCount := 1000000

	remoting.Start("127.0.0.1:8081")

	props := actor.
		FromProducer(newLocalActor(&wg, messageCount)).
		WithMailbox(actor.NewBoundedMailbox(1000, 1000))

	pid := actor.Spawn(props)

	remote := actor.NewPID("127.0.0.1:8080", "remote")
	remote.
		AskFuture(&messages.StartRemote{}, 5*time.Second).
		Wait()

	wg.Add(1)

	start := time.Now()
	log.Println("Starting to send")

	message := &messages.Ping{}
	for i := 0; i < messageCount; i++ {
		remote.Request(message, pid)
	}

	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("Elapsed %s", elapsed)

	x := int(float32(messageCount*2) / (float32(elapsed) / float32(time.Second)))
	log.Printf("Msg per sec %v", x)

	// f, err := os.Create("memprofile")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// pprof.WriteHeapProfile(f)
	// f.Close()
}
