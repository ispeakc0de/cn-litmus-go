package types

import (
	clientTypes "k8s.io/apimachinery/pkg/types"
)

// ExperimentDetails is for collecting all the experiment-related details
type ExperimentDetails struct {
	ExperimentName   string
	EngineName       string
	ChaosDuration    int
	ChaosInterval    int
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
	Sequence         string
	SourceDir        string
	DestinationDir   string
	ScriptFileName   string
	VMNames          string
	VMUserName       string
	VMPassword       string
	ScriptENVs       string
}
