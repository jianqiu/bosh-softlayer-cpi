package vm

import (
	"strconv"

	bosherr "bosh/errors"
	boshlog "bosh/logger"

	sl "github.com/maximilien/softlayer-go/softlayer"
	sldatatypes "github.com/maximilien/softlayer-go/data_types"

	bslcstem "github.com/maximilien/bosh-softlayer-cpi/softlayer/stemcell"
)

const softLayerCreatorLogTag = "SoftLayerCreator"

type SoftLayerCreator struct {
	softLayerClient        sl.Client
	agentEnvServiceFactory AgentEnvServiceFactory

	agentOptions AgentOptions
	logger       boshlog.Logger
}

func NewSoftLayerCreator(
	softLayerClient sl.Client,
	agentEnvServiceFactory AgentEnvServiceFactory,
	agentOptions AgentOptions,
	logger boshlog.Logger,
) SoftLayerCreator {
	return SoftLayerCreator{
		softLayerClient:        softLayerClient,
		agentEnvServiceFactory: agentEnvServiceFactory,

		agentOptions: agentOptions,
		logger:       logger,
	}
}

func (c SoftLayerCreator) Create(agentID string, stemcell bslcstem.Stemcell, networks Networks, env Environment) (VM, error) {
	virtualGuestTemplate := sldatatypes.SoftLayer_Virtual_Guest_Template{
		//Fill in necessary info from CloudProperties here
	}

	virtualGuestService, err := c.softLayerClient.GetSoftLayer_Virtual_Guest_Service()
	if err != nil {
		return SoftLayerVM{}, bosherr.WrapError(err, "Creating VirtualGuestService from SoftLayer client")
	}

	virtualGuest, err := virtualGuestService.CreateObject(virtualGuestTemplate)
	if err != nil {
		return SoftLayerVM{}, bosherr.WrapError(err, "Creating VirtualGuest from SoftLayer client")
	}

	agentEnv := NewAgentEnvForVM(agentID, strconv.Itoa(virtualGuest.Id), networks, env, c.agentOptions)

	agentEnvService := c.agentEnvServiceFactory.New()

	err = agentEnvService.Update(agentEnv)
	if err != nil {
		return SoftLayerVM{}, bosherr.WrapError(err, "Updating VM agent env")
	}

	vm := NewSoftLayerVM(
		virtualGuest.Id,
		c.softLayerClient,
		agentEnvService,
		c.logger,
	)

	return vm, nil
}

func (c SoftLayerCreator) resolveNetworkIP(networks Networks) (string, error) {
	var network Network

	switch len(networks) {
	case 0:
		return "", bosherr.New("Expected exactly one network; received zero")
	case 1:
		network = networks.First()
	default:
		return "", bosherr.New("Expected exactly one network; received multiple")
	}

	if network.IsDynamic() {
		return "", nil
	}

	return network.IP, nil
}