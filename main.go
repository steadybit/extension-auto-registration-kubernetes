package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-auto-registration-kubernetes/autoregistration"
	"github.com/steadybit/extension-auto-registration-kubernetes/client"
	"github.com/steadybit/extension-auto-registration-kubernetes/config"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extlogging"
	"github.com/steadybit/extension-kit/extruntime"
	"strconv"
	"time"
)

func main() {
	stopCh := make(chan struct{})
	defer close(stopCh)

	extlogging.InitZeroLog()
	extbuild.PrintBuildInformation()
	extruntime.LogRuntimeInformation(zerolog.DebugLevel)
	config.ParseConfiguration()
	initKlogBridge(config.Config.LogKubernetesHttpRequests)

	httpClientAgent := resty.New()
	httpClientAgent.BaseURL = "http://localhost:" + strconv.Itoa(config.Config.AgentPort)
	httpClientAgent.SetDisableWarn(true)

	k8sClient := client.PrepareClient(stopCh)

	//Sleep before first discovery to give the agent time to start
	log.Info().Float64("seconds", config.Config.InitialDelay.Seconds()).Msg("Initial delay before starting the discovery.")
	time.Sleep(config.Config.InitialDelay)
	autoregistration.UpdateAgentExtensions(httpClientAgent, k8sClient)

	// Wait indefinitely
	select {}
}
