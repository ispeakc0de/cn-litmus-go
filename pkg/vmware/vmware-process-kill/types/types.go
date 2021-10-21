package types

import (
	clientTypes "k8s.io/apimachinery/pkg/types"
)

// ADD THE ATTRIBUTES OF YOUR CHOICE HERE
// FEW MENDATORY ATTRIBUTES ARE ADDED BY DEFAULT

// ExperimentDetails is for collecting all the experiment-related details
type ExperimentDetails struct {
	ExperimentName   string
	EngineName       string
	ChaosDuration    int
	RampTime         int
	ChaosLib         string
	AppNS            string
	AppLabel         string
	AppKind          string
	AuxiliaryAppInfo string
	ChaosUID         clientTypes.UID
	InstanceID       string
	ChaosNamespace   string
	ChaosPodName     string
	Timeout          int
	Delay            int
	TargetContainer  string
	ProcessIds       string
	VMName           string
	VMUserName       string
	VMPassword       string
}
