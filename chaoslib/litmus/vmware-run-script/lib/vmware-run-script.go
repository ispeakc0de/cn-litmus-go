package lib

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chaosnative/litmus-go/pkg/cloud/vmware"
	experimentTypes "github.com/chaosnative/litmus-go/pkg/vmware/vmware-run-script/types"
	clients "github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
)

var wg sync.WaitGroup

// PrepareScriptChaos contains the prepration and injection steps for the experiment
func PrepareScriptChaos(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	//get the target VM names list
	vmNameList := strings.Split(experimentsDetails.VMNames, ",")
	if len(vmNameList) == 0 {
		return errors.Errorf("no target vms found")
	}

	switch strings.ToLower(experimentsDetails.Sequence) {
	case "serial":
		if err := injectChaosInSerialMode(experimentsDetails, vmNameList, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	case "parallel":
		if err := injectChaosInParallelMode(experimentsDetails, vmNameList, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	default:
		return errors.Errorf("%v sequence is not supported", experimentsDetails.Sequence)
	}

	//Waiting for the ramp time after chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	return nil
}

//injectChaosInSerialMode will inject the script chaos in serial mode which means one after the other
func injectChaosInSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, vmNameList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	sourceFilePath := getFilePath(experimentsDetails.SourceDir, experimentsDetails.ScriptFileName)
	destinationFilePath := getFilePath(experimentsDetails.DestinationDir, experimentsDetails.ScriptFileName)

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on VM"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		for i, vmName := range vmNameList {

			//Uploading the script to the target VM
			log.Infof("[Chaos]: Uploading the script to %s VM", vmName)
			if err := vmware.UploadScript(sourceFilePath, destinationFilePath, vmName, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
				return errors.Errorf("failed to upload the script to %s vm, err: %s", vmName, err.Error())
			}

			wg.Add(1)

			// script execution is a blocking task, hence it is being carried out in a goroutine to allow the probes to be run simultaenously
			_, err := executeScript(vmName, experimentsDetails.DestinationDir, experimentsDetails.ScriptFileName, strconv.Itoa(experimentsDetails.Timeout), experimentsDetails.VMUserName, experimentsDetails.VMPassword)
			if <-err != nil {
				return errors.Errorf("failed to execute the script in %s vm: %s", vmName, (<-err).Error())
			}

			common.SetTargets(vmName, "injected", "Script", chaosDetails)

			// run the probes during chaos
			if len(resultDetails.ProbeDetails) != 0 && i == 0 {
				if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
					return err
				}
			}

			wg.Wait()

			common.SetTargets(vmName, "reverted", "Script", chaosDetails)

			//Wait for chaos duration
			log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
			common.WaitForDuration(experimentsDetails.ChaosInterval)
		}
	}

	return nil
}

//injectChaosInSerialMode will inject the script chaos in parallel mode which means all at once
func injectChaosInParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, vmNameList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	sourceFilePath := getFilePath(experimentsDetails.SourceDir, experimentsDetails.ScriptFileName)
	destinationFilePath := getFilePath(experimentsDetails.DestinationDir, experimentsDetails.ScriptFileName)

	for duration < experimentsDetails.ChaosDuration {
		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on VM"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}
	}

	for _, vmName := range vmNameList {

		//Uploading the script to the target VM
		log.Infof("[Chaos]: Uploading the script to %s VM", vmName)
		if err := vmware.UploadScript(sourceFilePath, destinationFilePath, vmName, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
			return errors.Errorf("failed to upload the script to %s vm, err: %s", vmName, err.Error())
		}
	}

	for _, vmName := range vmNameList {

		wg.Add(1)

		// script execution is a blocking task, hence it is being carried out in a goroutine to allow the probes to be run simultaenously
		_, err := executeScript(vmName, experimentsDetails.DestinationDir, experimentsDetails.ScriptFileName, strconv.Itoa(experimentsDetails.Timeout), experimentsDetails.VMUserName, experimentsDetails.VMPassword)
		if <-err != nil {
			return errors.Errorf("failed to execute the script in %s vm: %s", vmName, (<-err).Error())
		}

		common.SetTargets(vmName, "injected", "Script", chaosDetails)
	}

	// run the probes during chaos
	if len(resultDetails.ProbeDetails) != 0 {
		if err := probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
			return err
		}
	}

	wg.Wait()

	for _, vmName := range vmNameList {

		common.SetTargets(vmName, "reverted", "Script", chaosDetails)
	}

	//Wait for chaos interval
	log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
	common.WaitForDuration(experimentsDetails.ChaosInterval)

	return nil
}

// getFilePath returns the filepath for a given filename and its directory
func getFilePath(dirPath, fileName string) string {

	if dirPath[len(dirPath)-1:] == "/" {
		return dirPath + fileName
	}

	return dirPath + "/" + fileName
}

// executeScript performs the chaos script execution in the target VM as a goroutine
func executeScript(vmName, destDir, scriptName, timeout, vmUserName, vmPassword string) (<-chan string, <-chan error) {

	chanErr := make(chan error)
	chanOutput := make(chan string)

	go func() {

		//Running the script in the target VM
		log.Infof("[Chaos]: Running the script in %s VM", vmName)
		output, err := vmware.ExecuteScript(destDir, scriptName, timeout, vmName, vmUserName, vmPassword)

		chanErr <- err
		chanOutput <- output

		wg.Done()
	}()

	return chanOutput, chanErr
}
