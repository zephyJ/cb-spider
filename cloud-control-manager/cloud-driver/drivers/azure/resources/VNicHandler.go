package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"reflect"
)

type AzureVNicHandler struct {
	Region       idrv.RegionInfo
	Ctx          context.Context
	NicClient    *network.InterfacesClient
	SubnetClient *network.SubnetsClient
}

func setterVNic(ni network.Interface) *irs.VNicInfo {
	nic := &irs.VNicInfo{
		Id:           *ni.ID,
		Name:         *ni.Name,
		Status:       *ni.ProvisioningState,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: CBResourceGroupName}},
	}

	if !reflect.ValueOf(ni.InterfacePropertiesFormat.MacAddress).IsNil() {
		nic.MacAddress = *ni.MacAddress
	}
	if !reflect.ValueOf(ni.InterfacePropertiesFormat.VirtualMachine).IsNil() {
		nic.OwnedVMID = *ni.InterfacePropertiesFormat.VirtualMachine.ID
	}
	if !reflect.ValueOf(ni.NetworkSecurityGroup).IsNil() {
		nic.SecurityGroupIds = []string{*ni.NetworkSecurityGroup.ID}
	}

	return nic
}

func (vNicHandler *AzureVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {
	// Check VNic Exists
	vNic, _ := vNicHandler.NicClient.Get(vNicHandler.Ctx, CBResourceGroupName, vNicReqInfo.Name, "")
	if vNic.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network Interface with name %s already exist", vNicReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.VNicInfo{}, createErr
	}

	subnet, err := vNicHandler.SubnetClient.Get(vNicHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, vNicReqInfo.VNetName, "")

	var ipConfigArr []network.InterfaceIPConfiguration
	ipConfig := network.InterfaceIPConfiguration{
		Name: to.StringPtr("ipConfig1"),
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &subnet,
			PrivateIPAllocationMethod: "Dynamic",
		},
	}
	if vNicReqInfo.PublicIPid != "" {
		ipConfig.PublicIPAddress = &network.PublicIPAddress{
			ID: &vNicReqInfo.PublicIPid,
		}
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	createOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
		},
		Location: &vNicHandler.Region.Region,
	}

	if len(vNicReqInfo.SecurityGroupIds) != 0 {
		createOpts.NetworkSecurityGroup = &network.SecurityGroup{
			ID: &vNicReqInfo.SecurityGroupIds[0],
		}
	}

	future, err := vNicHandler.NicClient.CreateOrUpdate(vNicHandler.Ctx, CBResourceGroupName, vNicReqInfo.Name, createOpts)
	if err != nil {
		return irs.VNicInfo{}, err
	}
	err = future.WaitForCompletionRef(vNicHandler.Ctx, vNicHandler.NicClient.Client)
	if err != nil {
		return irs.VNicInfo{}, err
	}

	// 생성된 VNet 정보 리턴
	vNetInfo, err := vNicHandler.GetVNic(vNicReqInfo.Name)
	if err != nil {
		return irs.VNicInfo{}, err
	}
	return vNetInfo, nil
}

func (vNicHandler *AzureVNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	result, err := vNicHandler.NicClient.List(vNicHandler.Ctx, CBResourceGroupName)
	if err != nil {
		return nil, err
	}

	var vNicList []*irs.VNicInfo
	for _, vNic := range result.Values() {
		vNicInfo := setterVNic(vNic)
		vNicList = append(vNicList, vNicInfo)
	}
	return vNicList, nil
}

func (vNicHandler *AzureVNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	vNic, err := vNicHandler.NicClient.Get(vNicHandler.Ctx, CBResourceGroupName, vNicID, "")
	if err != nil {
		return irs.VNicInfo{}, err
	}

	vNicInfo := setterVNic(vNic)
	return *vNicInfo, nil
}

func (vNicHandler *AzureVNicHandler) DeleteVNic(vNicID string) (bool, error) {
	future, err := vNicHandler.NicClient.Delete(vNicHandler.Ctx, CBResourceGroupName, vNicID)
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(vNicHandler.Ctx, vNicHandler.NicClient.Client)
	if err != nil {
		return false, err
	}
	return true, err
}
