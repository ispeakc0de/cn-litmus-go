package lib

import (
	"strings"
	"time"

	"github.com/chaosnative/litmus-go/pkg/cloud/vmware"
	experimentTypes "github.com/chaosnative/litmus-go/pkg/vmware/vmware-process-kill/types"
	"github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
)

// PrepareProcessKill contains the prepration and injection steps for the experiment
func PrepareProcessKill(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	processIdList := strings.Split(experimentsDetails.ProcessIds, ",")
	if len(processIdList) == 0 {
		return errors.Errorf("no processes found")
	}

	injectChaos(experimentsDetails, processIdList, clients, resultDetails, eventsDetails, chaosDetails)

	//Waiting for the ramp time after chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	return nil
}

// injectChaos will inject the process kill chaos in serial mode which means one after the other
func injectChaos(experimentsDetails *experimentTypes.ExperimentDetails, processIdList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on VM processes"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		for i, processId := range processIdList {

			// Killing the process
			log.Infof("[Chaos]: Killing process %s", processId)
			if err := vmware.KillProcess(processId, experimentsDetails.VMName, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
				return errors.Errorf("failed to kill process %s, err: %s", processId, err.Error())
			}

			common.SetTargets(processId, "injected", "Process", chaosDetails)

			// Wait for the process to be killed
			log.Infof("[Wait]: Wait for process %s to be killed", processId)
			if err := vmware.WaitForProcessKill(processId, experimentsDetails.VMName, experimentsDetails.VMUserName, experimentsDetails.VMPassword, chaosDetails.Delay, chaosDetails.Timeout); err != nil {
				return errors.Errorf("unable to kill process %s, err: %s", processId, err.Error())
			}

			// run the probes during chaos
			if len(resultDetails.ProbeDetails) != 0 && i == 0 {
				if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
					return err
				}
			}

			if i != 0 {
				//Wait for chaos interval
				log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
				common.WaitForDuration(experimentsDetails.ChaosInterval)
			}
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	return nil
}
