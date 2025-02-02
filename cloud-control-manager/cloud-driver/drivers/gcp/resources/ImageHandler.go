package resources

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	compute "google.golang.org/api/compute/v1"
)

type GCPImageHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

/*
이미지를 생성할 때 GCP 같은 경우는 내가 생성한 이미지에서만 리스트를 가져 올 수 있다.
퍼블릭 이미지를 가져 올 수 없다.
가져올라면 다르게 해야 함.
Insert할때 필수 값
name, sourceDisk(sourceImage),storageLocations(배열 ex : ["asia"])
이미지를 어떻게 생성하는냐에 따라서 키 값이 변경됨
디스크, 스냅샷,이미지, 가상디스크, Cloud storage
1) Disk일 경우 :
	{"sourceDisk": "projects/mcloud-barista-251102/zones/asia-northeast1-b/disks/my-root-pd",}
2) Image일 경우 :
	{"sourceImage": "projects/mcloud-barista-251102/global/images/image-1",}



*/

func (imageHandler *GCPImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {

	return irs.ImageInfo{}, nil
}

func (imageHandler *GCPImageHandler) ListImage() ([]*irs.ImageInfo, error) {

	projectId := imageHandler.Credential.ProjectID

	list, err := imageHandler.Client.Images.List(projectId).Do()
	if err != nil {
		log.Fatal(err)
	}
	var imageList []*irs.ImageInfo
	for _, item := range list.Items {
		info := mappingImageInfo(item)
		imageList = append(imageList, &info)
	}

	spew.Dump(imageList)
	return imageList, err
}

func (imageHandler *GCPImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	projectId := imageHandler.Credential.ProjectID

	image, err := imageHandler.Client.Images.Get(projectId, imageID).Do()
	if err != nil {
		log.Fatal(err)
	}
	imageInfo := mappingImageInfo(image)
	return imageInfo, err
}

func (imageHandler *GCPImageHandler) DeleteImage(imageID string) (bool, error) {
	projectId := imageHandler.Credential.ProjectID

	res, err := imageHandler.Client.Images.Delete(projectId, imageID).Do()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res)
	return true, err
}

func mappingImageInfo(imageInfo *compute.Image) irs.ImageInfo {
	lArr := strings.Split(imageInfo.Licenses[0], "/")
	os := lArr[len(lArr)-1]
	imageList := irs.ImageInfo{
		Id:      strconv.FormatUint(imageInfo.Id, 10),
		Name:    imageInfo.Name,
		GuestOS: os,
		Status:  imageInfo.Status,
		KeyValueList: []irs.KeyValue{
			{"SourceType", imageInfo.SourceType},
			{"SelfLink", imageInfo.SelfLink},
			{"GuestOsFeature", imageInfo.GuestOsFeatures[0].Type},
			{"DiskSizeGb", strconv.FormatInt(imageInfo.DiskSizeGb, 10)},
		},
	}

	return imageList

}
