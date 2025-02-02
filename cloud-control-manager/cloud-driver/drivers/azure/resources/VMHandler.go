// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	"reflect"
	"strings"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type AzureVMHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	Client         *compute.VirtualMachinesClient
}

func (vmHandler *AzureVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// Check VM Exists
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, CBResourceGroupName, vmReqInfo.VMName, compute.InstanceView)
	if vm.ID != nil {
		errMsg := fmt.Sprintf("VirtualMachine with name %s already exist", vmReqInfo.VMName)
		createErr := errors.New(errMsg)
		return irs.VMInfo{}, createErr
	}

	vmOpts := compute.VirtualMachine{
		Location: &vmHandler.Region.Region,
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			HardwareProfile: &compute.HardwareProfile{
				VMSize: compute.VirtualMachineSizeTypes(vmReqInfo.VMSpecId),
			},
			StorageProfile: &compute.StorageProfile{
				ImageReference: &compute.ImageReference{
					ID: &vmReqInfo.ImageId,
				},
			},
			OsProfile: &compute.OSProfile{
				ComputerName:  &vmReqInfo.VMName,
				AdminUsername: to.StringPtr(CBVMUser),
				//AdminPassword: &vmReqInfo.VMUserPasswd,
				/*LinuxConfiguration: &compute.LinuxConfiguration{
					SSH: &compute.SSHConfiguration{
						PublicKeys: &[]compute.SSHPublicKey{
							{
								//Path: to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", vmReqInfo.VMUserId)),
								//KeyData: &sshKeyData,
							},
						},
					},
				},*/
			},
			NetworkProfile: &compute.NetworkProfile{
				NetworkInterfaces: &[]compute.NetworkInterfaceReference{
					{
						ID: &vmReqInfo.NetworkInterfaceId,
						NetworkInterfaceReferenceProperties: &compute.NetworkInterfaceReferenceProperties{
							Primary: to.BoolPtr(true),
						},
					},
				},
			},
		},
	}

	if vmReqInfo.KeyPairName == "" {
		vmOpts.OsProfile.AdminPassword = to.StringPtr(vmReqInfo.VMUserPasswd)
	} else {
		publicKey, err := GetPublicKey(vmHandler.CredentialInfo, vmReqInfo.KeyPairName)
		if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}
		vmOpts.OsProfile.LinuxConfiguration = &compute.LinuxConfiguration{
			SSH: &compute.SSHConfiguration{
				PublicKeys: &[]compute.SSHPublicKey{
					{
						Path:    to.StringPtr(fmt.Sprintf("/home/%s/.ssh/authorized_keys", CBVMUser)),
						KeyData: to.StringPtr(publicKey),
					},
				},
			},
		}
	}

	future, err := vmHandler.Client.CreateOrUpdate(vmHandler.Ctx, CBResourceGroupName, vmReqInfo.VMName, vmOpts)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	vm, err = vmHandler.Client.Get(vmHandler.Ctx, CBResourceGroupName, vmReqInfo.VMName, compute.InstanceView)
	if err != nil {
		cblogger.Error(err)
	}
	vmInfo := mappingServerInfo(vm)

	return vmInfo, nil
}

func (vmHandler *AzureVMHandler) SuspendVM(vmID string) error {
	future, err := vmHandler.Client.PowerOff(vmHandler.Ctx, CBResourceGroupName, vmID)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) ResumeVM(vmID string) error {
	future, err := vmHandler.Client.Start(vmHandler.Ctx, CBResourceGroupName, vmID)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) RebootVM(vmID string) error {
	future, err := vmHandler.Client.Restart(vmHandler.Ctx, CBResourceGroupName, vmID)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) TerminateVM(vmID string) error {
	future, err := vmHandler.Client.Delete(vmHandler.Ctx, CBResourceGroupName, vmID)
	//future, err := vmHandler.Client.Deallocate(vmHandler.Ctx, CBResourceGroupName, vmID)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	err = future.WaitForCompletionRef(vmHandler.Ctx, vmHandler.Client.Client)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	return nil
}

func (vmHandler *AzureVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, CBResourceGroupName)
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMStatusInfo{}, err
	}

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Values() {
		if s.InstanceView != nil {
			statusStr := getVmStatus(*s.InstanceView)
			status := irs.VMStatus(statusStr)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     *s.ID,
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		} else {
			vmIdArr := strings.Split(*s.ID, "/")
			vmName := vmIdArr[8]
			status, _ := vmHandler.GetVMStatus(vmName)
			vmStatusInfo := irs.VMStatusInfo{
				VmId:     *s.ID,
				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

func (vmHandler *AzureVMHandler) GetVMStatus(vmID string) (irs.VMStatus, error) {
	instanceView, err := vmHandler.Client.InstanceView(vmHandler.Ctx, CBResourceGroupName, vmID)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	// Get powerState, provisioningState
	vmStatus := getVmStatus(instanceView)
	return irs.VMStatus(vmStatus), nil
}

func (vmHandler *AzureVMHandler) ListVM() ([]*irs.VMInfo, error) {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
	serverList, err := vmHandler.Client.List(vmHandler.Ctx, CBResourceGroupName)
	if err != nil {
		cblogger.Error(err)
		return []*irs.VMInfo{}, err
	}

	var vmList []*irs.VMInfo
	for _, server := range serverList.Values() {
		vmInfo := mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *AzureVMHandler) GetVM(vmID string) (irs.VMInfo, error) {
	vm, err := vmHandler.Client.Get(vmHandler.Ctx, CBResourceGroupName, vmID, compute.InstanceView)
	if err != nil {
		return irs.VMInfo{}, err
	}

	vmInfo := mappingServerInfo(vm)
	return vmInfo, nil
}

func getVmStatus(instanceView compute.VirtualMachineInstanceView) string {
	var powerState, provisioningState string

	for _, stat := range *instanceView.Statuses {
		statArr := strings.Split(*stat.Code, "/")

		if statArr[0] == "PowerState" {
			powerState = statArr[1]
		} else if statArr[0] == "ProvisioningState" {
			provisioningState = statArr[1]
		}
	}

	// Set VM Status Info
	var vmState string
	if powerState != "" && provisioningState != "" {
		vmState = powerState + "(" + provisioningState + ")"
	} else if powerState != "" && provisioningState == "" {
		vmState = powerState
	} else if powerState == "" && provisioningState != "" {
		vmState = provisioningState
	} else {
		vmState = "-"
	}
	return vmState
}

func mappingServerInfo(server compute.VirtualMachine) irs.VMInfo {

	// Get Default VM Info
	vmInfo := irs.VMInfo{
		Name: *server.Name,
		Id:   *server.ID,
		Region: irs.RegionInfo{
			Region: *server.Location,
		},
		VMSpecId: string(server.VirtualMachineProperties.HardwareProfile.VMSize),
	}

	// Set VM Zone
	if server.Zones != nil {
		vmInfo.Region.Zone = (*server.Zones)[0]
	}

	// Set VM Image Info
	if reflect.ValueOf(server.StorageProfile.ImageReference.ID).IsNil() {
		imageRef := server.VirtualMachineProperties.StorageProfile.ImageReference
		vmInfo.ImageId = *imageRef.Publisher + ":" + *imageRef.Offer + ":" + *imageRef.Sku + ":" + *imageRef.Version
	} else {
		vmInfo.ImageId = *server.VirtualMachineProperties.StorageProfile.ImageReference.ID
	}

	// Set VNic Info
	niList := *server.NetworkProfile.NetworkInterfaces
	for _, ni := range niList {
		if ni.NetworkInterfaceReferenceProperties != nil {
			vmInfo.VirtualNetworkId = *ni.ID
		}
	}

	// Set GuestUser Id/Pwd
	if server.VirtualMachineProperties.OsProfile.AdminUsername != nil {
		vmInfo.VMUserId = *server.VirtualMachineProperties.OsProfile.AdminUsername
	}
	if server.VirtualMachineProperties.OsProfile.AdminPassword != nil {
		vmInfo.VMUserPasswd = *server.VirtualMachineProperties.OsProfile.AdminPassword
	}

	// Set BootDisk
	if server.VirtualMachineProperties.StorageProfile.OsDisk.Name != nil {
		vmInfo.VMBootDisk = *server.VirtualMachineProperties.StorageProfile.OsDisk.Name
	}

	return vmInfo
}
