package main

import _ "net/http/pprof"

import (
	"flag"
	"fmt"
	"github.com/daseinio/dasein-p2p/crypto/ed25519"
	"github.com/daseinio/dasein-p2p/examples/local_benchmark/messages"
	"github.com/daseinio/dasein-p2p/network"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"time"
)

var profile = flag.String("profile", "", "write cpu profile to file")
var port = flag.Uint("port", 3002, "port to listen on")
var receiver = "tcp://localhost:3001"

func main() {
	flag.Set("logtostderr", "true")

	go func() {
		log.Println(http.ListenAndServe("localhost:7070", nil))
	}()

	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		pprof.StopCPUProfile()
		os.Exit(0)
	}()

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	builder := network.NewBuilder()
	builder.SetAddress("tcp://localhost:" + strconv.Itoa(int(*port)))
	builder.SetKeys(ed25519.RandomKeyPair())

	net, err := builder.Build()
	if err != nil {
		panic(err)
	}

	go net.Listen()
	net.Bootstrap(receiver)

	time.Sleep(500 * time.Millisecond)

	fmt.Printf("Spamming messages to %s...\n", receiver)

	client, err := net.Client(receiver)
	if err != nil {
		panic(err)
	}

	for {
		err = client.Tell(&messages.BasicMessage{})
		if err != nil {
			panic(err)
		}
	}
}
