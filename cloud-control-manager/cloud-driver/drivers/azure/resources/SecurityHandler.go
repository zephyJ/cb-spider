package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureSecurityHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *network.SecurityGroupsClient
}

func setterSec(securityGroup network.SecurityGroup) *irs.SecurityInfo {
	security := &irs.SecurityInfo{
		Id:           *securityGroup.ID,
		Name:         *securityGroup.Name,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: CBResourceGroupName}},
	}

	var securityRuleArr []irs.SecurityRuleInfo
	for _, sgRule := range *securityGroup.SecurityRules {
		ruleInfo := irs.SecurityRuleInfo{
			FromPort:   *sgRule.SourcePortRange,
			ToPort:     *sgRule.DestinationPortRange,
			IPProtocol: fmt.Sprint(sgRule.Protocol),
			Direction:  fmt.Sprint(sgRule.Direction),
		}

		securityRuleArr = append(securityRuleArr, ruleInfo)
	}
	security.SecurityRules = &securityRuleArr

	return security
}

func (securityHandler *AzureSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// Check SecurityGroup Exists
	security, _ := securityHandler.Client.Get(securityHandler.Ctx, CBResourceGroupName, securityReqInfo.Name, "")
	if security.ID != nil {
		errMsg := fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.SecurityInfo{}, createErr
	}

	var sgRuleList []network.SecurityRule
	var priorityNum int32
	for idx, rule := range *securityReqInfo.SecurityRules {
		priorityNum = int32(300 + idx*100)
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(fmt.Sprintf("%s-rules-%d", securityReqInfo.Name, idx+1)),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr(rule.FromPort),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr(rule.ToPort),
				Protocol:                 network.SecurityRuleProtocol(rule.IPProtocol),
				Access:                   network.SecurityRuleAccess("Allow"),
				Priority:                 to.Int32Ptr(priorityNum),
				Direction:                network.SecurityRuleDirection(rule.Direction),
			},
		}
		sgRuleList = append(sgRuleList, sgRuleInfo)
	}

	createOpts := network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &sgRuleList,
		},
		Location: &securityHandler.Region.Region,
	}

	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, CBResourceGroupName, securityReqInfo.Name, createOpts)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(securityReqInfo.Name)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	result, err := securityHandler.Client.List(securityHandler.Ctx, CBResourceGroupName)
	if err != nil {
		return nil, err
	}

	var securityList []*irs.SecurityInfo
	for _, security := range result.Values() {
		securityInfo := setterSec(security)
		securityList = append(securityList, securityInfo)
	}
	return securityList, nil
}

func (securityHandler *AzureSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	security, err := securityHandler.Client.Get(securityHandler.Ctx, CBResourceGroupName, securityID, "")
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := setterSec(security)
	return *securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	future, err := securityHandler.Client.Delete(securityHandler.Ctx, CBResourceGroupName, securityID)
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
