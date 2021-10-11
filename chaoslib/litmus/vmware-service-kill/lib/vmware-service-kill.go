package lib

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
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

var inject, abort chan os.Signal

// PrepareServiceKill facilitates chaos prepration and injection as per the value of SelfHealingServices
func PrepareServiceKill(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	switch experimentsDetails.SelfHealingServices {
	case "true":
		if err := serviceKillAndCheck(experimentsDetails, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	case "false":
		if err := serviceKillAndRestart(experimentsDetails, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	default:
		return errors.Errorf("%s value is not supported for SELF_HEALING_SERVICES", experimentsDetails.SelfHealingServices)
	}

	return nil
}

// serviceKillAndCheck executes chaos prepration and injection steps for self-healing services
func serviceKillAndCheck(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	//get the service names list
	serviceNamesList := strings.Split(experimentsDetails.ServiceNames, ",")
	if len(serviceNamesList) == 0 {
		return errors.Errorf("no service names found")
	}

	switch strings.ToLower(experimentsDetails.Sequence) {
	case "serial":
		if err := injectChaosInSerialMode(experimentsDetails, serviceNamesList, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	case "parallel":
		if err := injectChaosInParallelMode(experimentsDetails, serviceNamesList, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
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

// serviceKillAndRestart executes chaos prepration and injection steps for non self-healing services
func serviceKillAndRestart(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	// inject channel is used to transmit signal notifications.
	inject = make(chan os.Signal, 1)
	// Catch and relay certain signal(s) to inject channel.
	signal.Notify(inject, os.Interrupt, syscall.SIGTERM)

	// abort channel is used to transmit signal notifications.
	abort = make(chan os.Signal, 1)
	// Catch and relay certain signal(s) to abort channel.
	signal.Notify(abort, os.Interrupt, syscall.SIGTERM)

	//Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	//get the service names list
	serviceNamesList := strings.Split(experimentsDetails.ServiceNames, ",")
	if len(serviceNamesList) == 0 {
		return errors.Errorf("no service names found")
	}

	select {
	case <-inject:
		// stopping the chaos execution, if abort signal recieved
		os.Exit(0)
	default:

		// watching for the abort signal and revert the chaos
		go AbortWatcher(experimentsDetails, serviceNamesList, abort, chaosDetails)

		switch strings.ToLower(experimentsDetails.Sequence) {
		case "serial":
			if err := injectChaosInSerialMode(experimentsDetails, serviceNamesList, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
				return err
			}
		case "parallel":
			if err := injectChaosInParallelMode(experimentsDetails, serviceNamesList, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
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
	}

	return nil
}

// injectChaosInSerialMode stops the services in serial mode i.e. one by one
func injectChaosInSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, serviceNamesList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {
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

			if experimentsDetails.SelfHealingServices == "true" {

				//Wait for service to start
				log.Infof("[Wait]: Wait for %s service to start", serviceName)
				if err := vmware.WaitForServiceStart(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
					return errors.Errorf("%s service failed to stop, %s", serviceName, err)
				}
			} else {

				//Wait for chaos duration
				log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
				common.WaitForDuration(experimentsDetails.ChaosInterval)

				// Get service state
				serviceState, err := vmware.GetServiceState(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword)
				if err != nil {
					return errors.Errorf("failed to get %s service state, %s", serviceState, err)
				}

				switch serviceState {
				case "active":
					log.Infof("[Skip]: %s service is already active", serviceName)
				default:
					// Start the service
					log.Info("[Chaos]: Starting %s service")
					if err := vmware.StartService(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
						return errors.Errorf("unable to start %s service, %s", serviceName, err)
					}

					//Wait for service to start
					log.Infof("[Wait]: Wait for %s service to start")
					if err := vmware.WaitForServiceStop(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
						return errors.Errorf("unable to start %s service, %s", serviceName, err)
					}
				}
			}

			common.SetTargets(serviceName, "reverted", "Service", chaosDetails)
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	return nil
}

// injectChaosInParallelMode stops the services in parallel mode i.e. all at once
func injectChaosInParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, serviceNamesList []string, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

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

		if experimentsDetails.SelfHealingServices == "true" {
			for _, serviceName := range serviceNamesList {

				//Wait for service to start
				log.Infof("[Wait]: Wait for %s service to start", serviceName)
				if err := vmware.WaitForServiceStart(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
					return errors.Errorf("%s service failed to stop, %s", serviceName, err)
				}

				common.SetTargets(serviceName, "reverted", "Service", chaosDetails)
			}
		} else {
			//Wait for chaos interval
			log.Infof("[Wait]: Waiting for the chaos interval of %vs", experimentsDetails.ChaosInterval)
			common.WaitForDuration(experimentsDetails.ChaosInterval)

			for _, serviceName := range serviceNamesList {

				// Get the service state
				serviceState, err := vmware.GetServiceState(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword)
				if err != nil {
					return errors.Errorf("failed to get %s service state, %s", serviceState, err)
				}

				switch serviceState {
				case "active":
					log.Infof("[Skip]: %s service is already active", serviceName)
				default:
					// Start the service
					log.Info("[Chaos]: Starting %s service")
					if err := vmware.StartService(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword); err != nil {
						return errors.Errorf("unable to start %s service, %s", serviceName, err)
					}

					//Wait for service to start
					log.Infof("[Wait]: Wait for %s service to start")
					if err := vmware.WaitForServiceStop(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
						return errors.Errorf("unable to start %s service, %s", serviceName, err)
					}
				}

				common.SetTargets(serviceName, "reverted", "Service", chaosDetails)
			}
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	return nil
}

// AbortWatcher will watching for the abort signal and revert the chaos
func AbortWatcher(experimentsDetails *experimentTypes.ExperimentDetails, serviceNamesList []string, abort chan os.Signal, chaosDetails *types.ChaosDetails) {
	<-abort

	log.Info("[Abort]: Chaos Revert Started")

	for _, serviceName := range serviceNamesList {

		// Getting the service state
		serviceState, err := vmware.GetServiceState(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword)
		if err != nil {
			log.Errorf("failed to get the service state, %s", err.Error())
		}

		if serviceState != "active" {

			//Wait for the service to completely stop
			//We first wait for the service to get in inactive state then we are restarting it.
			log.Infof("[Abort]: Wait for %s service to completely stop", serviceName)

			if err := vmware.WaitForServiceStop(experimentsDetails.VcenterServer, experimentsDetails.VMName, serviceName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword, experimentsDetails.Delay, experimentsDetails.Timeout); err != nil {
				log.Errorf("unable to stop the service, err: %v", err)
			}

			//Starting the service
			log.Infof("[Abort]: Starting the %s service", serviceName)

			err := vmware.StartService(serviceName, experimentsDetails.VMName, experimentsDetails.Datacenter, experimentsDetails.VMUserName, experimentsDetails.VMPassword)
			if err != nil {
				log.Errorf("unable to start service %s during abort, %s", err)
			}
		}

		common.SetTargets(serviceName, "reverted", "Service", chaosDetails)
	}

	log.Info("[Abort]: Chaos Revert Completed")
	os.Exit(1)
}
