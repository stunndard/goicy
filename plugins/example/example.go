// Command example_plugin is an example of a very simple plugin.
//
// example_plugin provides two APIs that communicate via JSON-RPC.  It is
// expected to be started by example_master.
package main

import (
	"log"
	"net/rpc/jsonrpc"

	"github.com/natefinch/pie"
)

type api struct{}

type Info struct {
	Name string
	Type int
}

func main() {
	p := pie.NewProvider()
	if err := p.RegisterName("Plugin", api{}); err != nil {
		log.Fatalf("failed to register Plugin: %s", err)
	}

	p.ServeCodec(jsonrpc.NewServerCodec)
}

func (api) Init(request string, response *Info) error {
	log.Printf("got call for Init " + request)

	*response = Info{"example", 1}
	return nil
}

func (api) First(request string, response *[]string) error {
	log.Printf("got call for First " + request)

	*response = []string{
		"http://very.complicated.url.to.play/yes/yes/of.course.mp4",
		"the title of the song",
	}
	return nil
}

func (api) Next(request string, response *[]string) error {
	log.Printf("got call for Next " + request)

	*response = []string{
		"plugin://no.more",
		"the title",
	}
	return nil
}


