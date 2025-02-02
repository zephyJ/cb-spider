package resources

import (
	"context"
	"log"
	"strconv"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
)

// GCP는 networkInterface 를 따로 핸들링 하는 API는 갖고 있지 않다.
// 따라서 Instance에서 추출해서 써야 함.
// securityGroup는 나중에 vnet에 할당 또는 tag를 달아서 태그에 할당하는 형태
// networkInterface name은 자동으로 생성됨 지정 못함.
type GCPVNicHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

func (vNicHandler *GCPVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	return irs.VNicInfo{}, nil
}

func (vNicHandler *GCPVNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	projectId := vNicHandler.Credential.ProjectID
	zone := vNicHandler.Region.Zone
	res, err := vNicHandler.Client.Instances.List(projectId, zone).Do()
	var vNicInfo []*irs.VNicInfo
	for _, item := range res.Items {
		info := vNicHandler.mappingNetworkInfo(item)
		vNicInfo = append(vNicInfo, &info)
	}
	return vNicInfo, err
}

func (vNicHandler *GCPVNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	projectId := vNicHandler.Credential.ProjectID
	zone := vNicHandler.Region.Zone

	res, err := vNicHandler.Client.Instances.Get(projectId, zone, vNicID).Do()
	if err != nil {
		log.Fatal(err)
	}
	vNicInfo := irs.VNicInfo{
		Id:        strconv.FormatUint(res.Id, 10),
		Name:      res.NetworkInterfaces[0].Name,
		PublicIP:  res.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		OwnedVMID: strconv.FormatUint(res.Id, 10),
		Status:    res.Status, //nic 상태를 알 수 있는 것이 없어서 Instance의 상태값을 가져다 넣어줌
		KeyValueList: []irs.KeyValue{
			{"Network", res.NetworkInterfaces[0].Network},
			{"NetworkIP", res.NetworkInterfaces[0].NetworkIP},
			{"PublicIPName", res.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", res.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"Network", res.NetworkInterfaces[0].Network},
		},
	}

	return vNicInfo, err
}

func (vNicHandler *GCPVNicHandler) DeleteVNic(vNicID string) (bool, error) {
	//  networkInterface를 삭제 하는 API 및 기능이 없음
	return true, nil
}

func (*GCPVNicHandler) mappingNetworkInfo(res *compute.Instance) irs.VNicInfo {
	netWorkInfo := irs.VNicInfo{
		Id:        strconv.FormatUint(res.Id, 10),
		Name:      res.NetworkInterfaces[0].Name,
		PublicIP:  res.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		OwnedVMID: strconv.FormatUint(res.Id, 10),
		Status:    res.Status, //nic 상태를 알 수 있는 것이 없어서 Instance의 상태값을 가져다 넣어줌
		KeyValueList: []irs.KeyValue{
			{"Network", res.NetworkInterfaces[0].Network},
			{"NetworkIP", res.NetworkInterfaces[0].NetworkIP},
			{"PublicIPName", res.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", res.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"Network", res.NetworkInterfaces[0].Network},
		},
	}

	return netWorkInfo

}
