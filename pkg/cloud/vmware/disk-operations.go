package vmware

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	vmwareLib "github.com/litmuschaos/litmus-go/pkg/cloud/vmware"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/utils/retry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// GetDisks returns the list of ids of disks attached to a given VM
func GetDisks(vcenterServer, vmId, cookie string) ([]string, error) {

	type DiskList struct {
		MsgValue []struct {
			MsgDisk string `json:"disk"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm/"+vmId+"/hardware/disk/", nil)
	if err != nil {
		return nil, errors.Errorf(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		json.Unmarshal(body, &errorResponse)
		return nil, errors.Errorf("error during the fetching of disks: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var diskDetails DiskList
	json.Unmarshal(body, &diskDetails)

	var diskIdList []string
	for _, disk := range diskDetails.MsgValue {
		diskIdList = append(diskIdList, disk.MsgDisk)
	}

	return diskIdList, nil
}

// DiskDetach will detach a disk from a VM
func DiskDetach(vcenterServer, appVMMoid, diskId, cookie string) error {

	req, err := http.NewRequest("DELETE", "https://"+vcenterServer+"/rest/vcenter/vm/"+appVMMoid+"/hardware/disk/"+diskId, nil)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errorResponse vmwareLib.ErrorResponse

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf(err.Error())
		}

		json.Unmarshal(body, &errorResponse)

		return errors.Errorf("error during disk detachment: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	log.InfoWithValues("Detached disk having:", logrus.Fields{
		"VM ID":   appVMMoid,
		"Disk ID": diskId,
	})

	return nil
}

// DiskAttach will attach a disk to a VM
func DiskAttach(vcenterServer, appVMMoid, diskPath, cookie string) error {

	type AttachDiskResponse struct {
		MsgValue string `json:"value"`
	}

	jsonString := fmt.Sprintf(`{"spec":{"backing":{"type":"VMDK_FILE","vmdk_file":"%s"}}}`, diskPath)

	req, err := http.NewRequest("POST", "https://"+vcenterServer+"/rest/vcenter/vm/"+appVMMoid+"/hardware/disk", bytes.NewBuffer([]byte(jsonString)))
	if err != nil {
		return errors.Errorf(err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Errorf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse vmwareLib.ErrorResponse

		json.Unmarshal(body, &errorResponse)

		return errors.Errorf("error during disk attachment: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var response AttachDiskResponse
	json.Unmarshal(body, &response)

	log.InfoWithValues("Attached disk having:", logrus.Fields{
		"VM ID":   appVMMoid,
		"Disk ID": response.MsgValue,
	})

	return nil
}

// GetDiskPath returns the path of the VMDK disk file for a given disk id
func GetDiskPath(vcenterServer, appVMMoid, diskId, cookie string) (string, error) {

	type DiskInfo struct {
		MsgValue struct {
			MsgBacking struct {
				MsgVMDKFile string `json:"vmdk_file"`
			} `json:"backing"`
		} `json:"value"`
	}

	req, err := http.NewRequest("GET", "https://"+vcenterServer+"/rest/vcenter/vm/"+appVMMoid+"/hardware/disk/"+diskId, nil)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Errorf(err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse vmwareLib.ErrorResponse

		json.Unmarshal(body, &errorResponse)

		return "", errors.Errorf("error during disk information fetch: %s", errorResponse.MsgValue.MsgMessages[0].MsgDefaultMessage)
	}

	var diskInfo DiskInfo
	json.Unmarshal(body, &diskInfo)

	return diskInfo.MsgValue.MsgBacking.MsgVMDKFile, nil
}

// WaitForDiskDetachment will wait for the disk to completely detach from the VM
func WaitForDiskDetachment(vcenterServer, appVMMoid, diskId, cookie string, delay, timeout int) error {

	log.Info("[Status]: Checking disk status for detachment")
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			diskState, err := GetDiskState(vcenterServer, appVMMoid, diskId, cookie)
			if err != nil {
				return errors.Errorf("failed to get the disk state")
			}

			if diskState != "detached" {
				log.Infof("[Info]: The disk state is %v", diskState)
				return errors.Errorf("disk is not yet in detached state")
			}

			log.Infof("[Info]: The disk state is %v", diskState)
			return nil
		})
}

// WaitForDiskAttachment will wait for the disk to get attached to the VM
func WaitForDiskAttachment(vcenterServer, appVMMoid, diskId, cookie string, delay, timeout int) error {

	log.Info("[Status]: Checking disk status for attachment")
	return retry.
		Times(uint(timeout / delay)).
		Wait(time.Duration(delay) * time.Second).
		Try(func(attempt uint) error {

			diskState, err := GetDiskState(vcenterServer, appVMMoid, diskId, cookie)
			if err != nil {
				return errors.Errorf("failed to get the disk status")
			}

			if diskState != "attached" {
				log.Infof("[Info]: The disk state is %v", diskState)
				return errors.Errorf("disk is not yet in attached state")
			}

			log.Infof("[Info]: The disk state is %v", diskState)
			return nil
		})
}
