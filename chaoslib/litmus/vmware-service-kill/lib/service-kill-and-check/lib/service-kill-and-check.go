package lib

import (
	"time"

	"github.com/chaosnative/litmus-go/pkg/cloud/vmware"
	experimentTypes "github.com/chaosnative/litmus-go/pkg/vmware/vmware-service-kill/types"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
)

// ServiceKillAndCheckSerialMode kills a service and checks if it restarts by itself in serial mode
func ServiceKillAndCheckSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, serviceNamesList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on service"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		for i, serviceName := range serviceNamesList {

			//Stopping the service
			log.Infof("[Chaos]: Stopping %s service", serviceName)
			if err := vmware.StopService(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
				return errors.Errorf("unable to stop %s service, %s", serviceName, err)
			}

			common.SetTargets(serviceName, "injected", "Service", chaosDetails)

			//Wait for service to stop
			log.Infof("[Wait]: Wait for %s service to stop", serviceName)
			if err := vmware.WaitForServiceStop(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				return errors.Errorf("%s service failed to stop, %s", serviceName, err)
			}

			// run the probes during chaos
			if len(resultDetails.ProbeDetails) != 0 && i == 0 {
				if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
					return err
				}
			}

			//Wait for service to start
			log.Infof("[Wait]: Wait for %s service to start", serviceName)
			if err := vmware.WaitForServiceStart(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				return errors.Errorf("%s service failed to stop, %s", serviceName, err)
			}

			common.SetTargets(serviceName, "reverted", "Service", chaosDetails)
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	return nil
}

// ServiceKillAndCheckParallelMode kills a service and checks if it restarts by itself in parallel mode
func ServiceKillAndCheckParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, serviceNamesList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on service"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		for _, serviceName := range serviceNamesList {

			//Stopping the service
			log.Infof("[Chaos]: Stopping %s service", serviceName)
			if err := vmware.StopService(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
				return errors.Errorf("unable to stop %s service, %s", serviceName, err)
			}

			common.SetTargets(serviceName, "injected", "Service", chaosDetails)
		}

		for _, serviceName := range serviceNamesList {

			//Wait for service to stop
			log.Infof("[Wait]: Wait for %s service to stop", serviceName)
			if err := vmware.WaitForServiceStop(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				return errors.Errorf("%s service failed to stop, %s", serviceName, err)
			}
		}

		// run the probes during chaos
		if len(resultDetails.ProbeDetails) != 0 {
			if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
				return err
			}
		}

		for _, serviceName := range serviceNamesList {

			//Wait for service to start
			log.Infof("[Wait]: Wait for %s service to start", serviceName)
			if err := vmware.WaitForServiceStart(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				return errors.Errorf("%s service failed to stop, %s", serviceName, err)
			}

			common.SetTargets(serviceName, "reverted", "Service", chaosDetails)
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	return nil
}
