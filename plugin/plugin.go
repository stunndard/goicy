package plugin

import (
	"errors"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"strings"

	"github.com/natefinch/pie"
	"github.com/stunndard/goicy/logger"
)

type plug struct {
	client *rpc.Client
}

/*
const (
	outputName = iota
	outputBuffer
)
*/

type Info struct {
	Name string
	Type int
}

var (
	pl plug
)

func Load(pluginPath string) (err error) {

	pluginParams := strings.Split(pluginPath[9:], "|")
	if len(pluginParams) < 2 {
		return errors.New("invalid plugin params: " + pluginPath)
	}
	pluginFile := "plugin_" + pluginParams[0] + ".exe"

	client, err := pie.StartProviderCodec(jsonrpc.NewClientCodec, os.Stderr, pluginFile)
	if err != nil {
		return errors.New("error starting plugin: " + pluginFile + " " + err.Error())
	}

	pl = plug{client}

	pluginInfo, err := pl.Init(pluginParams[1])
	if err != nil {
		return errors.New("error calling Init: " + err.Error())
	}

	logger.Log(fmt.Sprintf("Response from plugin: %q", pluginInfo), logger.LogDebug)
	return err
}

/*
func Unload() {
	pl.client.Close()
}
*/

func First() (file, metadata string, err error) {
	fileMetadata, err := pl.First()
	return fileMetadata[0], fileMetadata[1], err
}

func Next() (file, metadata string, err error) {
	fileMetadata, err := pl.Next()
	return fileMetadata[0], fileMetadata[1], err
}

func (p plug) Init(pluginParams string) (result Info, err error) {
	err = p.client.Call("Plugin.Init", pluginParams, &result)
	return result, err
}

func (p plug) First() (result []string, err error) {
	err = p.client.Call("Plugin.First", nil, &result)
	return result, err
}

func (p plug) Next() (result []string, err error) {
	err = p.client.Call("Plugin.Next", nil, &result)
	return result, err
}
