/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lvm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/container-storage-interface/spec/lib/go/csi"
	volume "github.com/kata-containers/kata-containers/src/runtime/pkg/direct-volume"
	"github.com/kubernetes-csi/drivers/pkg/csi-common"
	runner "github.com/kubeservice-stack/kata-local-csi-driver/pkg/csi"
	"github.com/kubeservice-stack/kata-local-csi-driver/pkg/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	log "k8s.io/klog/v2"

	k8smount "k8s.io/utils/mount"
)

const (
	// NsenterCmd is the nsenter command
	NsenterCmd = "/nsenter --mount=/proc/1/ns/mnt"
	// VgNameTag is the vg name tag
	VgNameTag = "vgName"
	// PvTypeTag is the pv type tag
	PvTypeTag = "pvType"
	// FsTypeTag is the fs type tag
	FsTypeTag = "fsType"
	// LvmTypeTag is the lvm type tag
	LvmTypeTag = "lvmType"
	// LocalDisk local disk
	LocalDisk = "localdisk"
	// CloudDisk cloud disk
	CloudDisk = "clouddisk"
	// LinearType linear type
	LinearType = "linear"
	// StripingType striping type
	StripingType = "striping"
	// DefaultFs default fs
	DefaultFs = "ext4"
	// DefaultNA default NodeAffinity
	DefaultNA = "true"
	DirectTag = "direct"
	// TopologyNodeKey tag
	TopologyNodeKey = "topology.lvmplugin.csi.kubeservice.cn/hostname"
)

type nodeServer struct {
	*csicommon.DefaultNodeServer
	nodeID     string
	mounter    utils.Mounter
	client     kubernetes.Interface
	k8smounter k8smount.Interface
	rn         runner.ExecRunner
	isDirect   bool
}

var (
	// DeviceChars is chars of a device
	DeviceChars = []string{"b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}
)

// NewNodeServer create a NodeServer object
func NewNodeServer(d *csicommon.CSIDriver, nodeID string) csi.NodeServer {
	cfg, err := clientcmd.BuildConfigFromFlags(utils.MasterURL, utils.Kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	return &nodeServer{
		DefaultNodeServer: csicommon.NewDefaultNodeServer(d),
		nodeID:            nodeID,
		mounter:           utils.NewMounter(),
		k8smounter:        k8smount.New(""),
		client:            kubeClient,
		rn:                runner.NewExecRunner(),
		isDirect:          false,
	}
}

func (ns *nodeServer) GetNodeID() string {
	return ns.nodeID
}

func (ns *nodeServer) addDirectVolume(volumePath, device, fsType string) error {
	mountInfo := volume.MountInfo{
		VolumeType: "block",
		Device:     device,
		FsType:     fsType,
	}

	mi, err := json.Marshal(mountInfo)
	if err != nil {
		log.Error("addDirectVolume - json.Marshal failed: ", err.Error())
		return status.Errorf(codes.Internal, "json.Marshal failed: %s", err.Error())
	}

	if err := volume.Add(volumePath, string(mi)); err != nil {
		log.Error("addDirectVolume - add direct volume failed: ", err.Error())
		return status.Errorf(codes.Internal, "add direct volume failed: %s", err.Error())
	}

	log.Infof("add direct volume done: %s %s", volumePath, string(mi))
	return nil
}

func (ns *nodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	log.V(4).Infof("NodePublishVolume: called with args %+v", *req)

	// Step 1: check
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: Volume ID not provided")
	}
	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: targetPath is empty")
	}
	log.Infof("NodePublishVolume: start to mount volume %s to target path %s", volumeID, targetPath)

	vgName := ""
	if _, ok := req.VolumeContext[VgNameTag]; ok {
		vgName = req.VolumeContext[VgNameTag]
	}
	if vgName == "" {
		return nil, status.Error(codes.InvalidArgument, "NodePublishVolume: must set vgName in volumeAttributes")
	}
	pvType := CloudDisk
	if _, ok := req.VolumeContext[PvTypeTag]; ok {
		pvType = req.VolumeContext[PvTypeTag]
	}
	lvmType := LinearType
	if _, ok := req.VolumeContext[LvmTypeTag]; ok {
		lvmType = req.VolumeContext[LvmTypeTag]
	}
	fsType := DefaultFs
	if _, ok := req.VolumeContext[FsTypeTag]; ok {
		fsType = req.VolumeContext[FsTypeTag]
	}
	log.Infof("NodePublishVolume: Starting to mount lvm at: %s, with vg: %s, with volume: %s, PV type: %s, LVM type: %s", targetPath, vgName, req.GetVolumeId(), pvType, lvmType)

	// check if the volume is a direct-assigned volume, direct volume will be used as virtio-blk
	ns.isDirect = false
	if val, ok := req.VolumeContext[DirectTag]; ok {
		var err error
		ns.isDirect, err = strconv.ParseBool(val)
		if err != nil {
			ns.isDirect = false
		}
	}

	// Step 3: check new create
	volumeNewCreated := false
	devicePath := filepath.Join("/dev/", vgName, volumeID)
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		volumeNewCreated = true
		err := ns.createVolume(ctx, volumeID, vgName, pvType, lvmType)
		if err != nil {
			return nil, status.Error(codes.Aborted, "vgName/pvType/lvmType/volumeID not found, err: "+err.Error())
		}
	}

	// Step 4: direct
	if ns.isDirect {
		if err := ns.addDirectVolume(targetPath, devicePath, fsType); err != nil {
			log.Error("addDirectVolume failed: ", err.Error())
			return nil, status.Errorf(codes.Internal, "addDirectVolume failed: %s", err.Error())
		}

		log.Infof("NodePublishVolume: add kata direct volume %s to %s successfully", volumeID, targetPath)
		return &csi.NodePublishVolumeResponse{}, nil
	}

	// Step 5: Mount
	isMnt, err := ns.mounter.IsMounted(targetPath)
	if err != nil {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			isMnt = false
		} else {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	exitFSType, err := checkFSType(devicePath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "check fs type err: %v", err)
	}
	if exitFSType == "" {
		log.Infof("The device %v has no filesystem, starting format: %v", devicePath, fsType)
		if err := formatDevice(devicePath, fsType); err != nil {
			return nil, status.Errorf(codes.Internal, "format fstype failed: err=%v", err)
		}
	}

	if !isMnt {
		var options []string
		if req.GetReadonly() {
			options = append(options, "ro")
		} else {
			options = append(options, "rw")
		}
		mountFlags := req.GetVolumeCapability().GetMount().GetMountFlags()
		options = append(options, mountFlags...)

		err = ns.mounter.Mount(devicePath, targetPath, fsType, options...)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		log.Infof("NodePublishVolume:: mount successful devicePath: %s, targetPath: %s, options: %v", devicePath, targetPath, options)
	}

	// xfs filesystem works on targetpath.
	if volumeNewCreated == false {
		if err := ns.resizeVolume(ctx, volumeID, vgName, targetPath); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	log.V(4).Infof("NodeUnpublishVolume: called with args %+v", *req)
	// Step 1: check
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeUnpublishVolume: Volume ID not provided")
	}
	targetPath := req.GetTargetPath()
	if targetPath == "" {
		return nil, status.Error(codes.Internal, "NodeUnpublishVolume: targetPath is empty")
	}
	log.Infof("NodeUnpublishVolume: start to umount target path %s for volume %s", targetPath, volumeID)

	// Step 2: umount
	if ns.isDirect {
		if err := volume.Remove(targetPath); err != nil {
			log.Errorf("NodeUnpublishVolume: direct volume remove failed: %s", err.Error())
		}
	}
	isMnt, err := ns.mounter.IsMounted(targetPath)
	if err != nil {
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return nil, status.Error(codes.NotFound, "TargetPath not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !isMnt {
		return &csi.NodeUnpublishVolumeResponse{}, nil
	}

	err = ns.mounter.Unmount(req.GetTargetPath())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (ns *nodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	log.V(4).Infof("NodeGetCapabilities: called with args %+v", *req)
	return &csi.NodeGetCapabilitiesResponse{Capabilities: runner.NodeCaps}, nil
}

func (ns *nodeServer) NodeExpandVolume(ctx context.Context, req *csi.NodeExpandVolumeRequest) (
	*csi.NodeExpandVolumeResponse, error) {
	log.V(4).Infof("NodeExpandVolume: called with args %+v", *req)
	volumeID := req.VolumeId
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeExpandVolume: Volume ID not provided")
	}
	targetPath := req.VolumePath
	if len(targetPath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "NodeExpandVolume: Target path not provided")
	}
	expectSize := req.CapacityRange.RequiredBytes

	if ns.isDirect {
		log.Warning("Warning: direct volume resize is to be implemented")
	} else {
		_, _, _, pv := ns.getPvInfo(volumeID)
		// Get lvm info
		vgName := utils.GetVGNameFromCsiPV(pv)
		if vgName == "" {
			return nil, status.Errorf(codes.Internal, "resizeVolume: Volume %s with vgname empty", pv.Name)
		}
		if err := ns.resizeVolume(ctx, volumeID, vgName, targetPath); err != nil {
			return nil, status.Errorf(codes.Internal, "NodeExpandVolume: Resize local volume %s with error: %s", volumeID, err.Error())
		}
	}
	log.Infof("NodeExpandVolume: Successful expand local volume: %v to %d", req.VolumeId, expectSize)
	return &csi.NodeExpandVolumeResponse{}, nil
}

func (ns *nodeServer) NodeGetInfo(ctx context.Context, req *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	return &csi.NodeGetInfoResponse{
		NodeId: ns.nodeID,
		// make sure that the driver works on this particular node only
		AccessibleTopology: &csi.Topology{
			Segments: map[string]string{
				TopologyNodeKey: ns.nodeID,
			},
		},
	}, nil
}

func (ns *nodeServer) resizeVolume(ctx context.Context, volumeID, vgName, targetPath string) error {
	pvSize, pvSizeByte, unit, _ := ns.getPvInfo(volumeID)
	devicePath := filepath.Join("/dev", vgName, volumeID)
	sizeCmd := fmt.Sprintf("%s lvdisplay --units B %s 2>&1 | grep 'LV Size' | awk '{print $3}'", NsenterCmd, devicePath)
	sizeStr, err := utils.Run(sizeCmd)
	if err != nil {
		return err
	}
	if sizeStr == "" {
		return status.Error(codes.Internal, "Get lvm size error")
	}
	sizeStr = strings.Split(sizeStr, ".")[0]
	sizeInt, err := strconv.ParseInt(strings.TrimSpace(sizeStr), 10, 64)
	if err != nil {
		return err
	}

	// if lvmsize equal/bigger than pv size, no do expand.
	if sizeInt >= pvSizeByte {
		return nil
	}
	log.Infof("NodeExpandVolume:: volumeId: %s, devicePath: %s, from size: %d, to Size: %d%s", volumeID, devicePath, sizeInt, pvSize, unit)

	// resize lvm volume
	// lvextend -L3G /dev/vgtest/lvm-5db74864-ea6b-11e9-a442-00163e07fb69
	resizeCmd := fmt.Sprintf("%s lvextend -L%d%s %s", NsenterCmd, pvSize, unit, devicePath)
	_, err = utils.Run(resizeCmd)
	if err != nil {
		return err
	}

	// use resizer to expand volume filesystem
	//resizer := resizefs.NewResizeFs(&k8smount.SafeFormatAndMount{Interface: ns.k8smounter, Exec: utilexec.New()})
	ok, err := ns.rn.ResizeFS(devicePath, targetPath)
	if err != nil {
		log.Errorf("NodeExpandVolume:: Resize Error, volumeId: %s, devicePath: %s, volumePath: %s, err: %s", volumeID, devicePath, targetPath, err.Error())
		return err
	}
	if !ok {
		log.Errorf("NodeExpandVolume:: Resize failed, volumeId: %s, devicePath: %s, volumePath: %s", volumeID, devicePath, targetPath)
		return status.Error(codes.Internal, "Fail to resize volume fs")
	}
	log.Infof("NodeExpandVolume:: resizefs successful volumeId: %s, devicePath: %s, volumePath: %s", volumeID, devicePath, targetPath)
	return nil
}

func (ns *nodeServer) getPvInfo(volumeID string) (int64, int64, string, *v1.PersistentVolume) {
	pv, err := ns.client.CoreV1().PersistentVolumes().Get(context.Background(), volumeID, metav1.GetOptions{})
	if err != nil {
		log.Errorf("lvcreate: fail to get pv, err: %v", err)
		return 0, 0, "", nil
	}
	pvQuantity := pv.Spec.Capacity["storage"]
	pvSizeByte := pvQuantity.Value()
	pvSize := pvSizeByte
	pvSizeGB := pvSize / (1024 * 1024 * 1024)

	if pvSizeGB == 0 {
		pvSizeMB := pvSize / (1024 * 1024)
		return pvSizeMB, pvSizeByte, "m", pv
	}
	return pvSizeGB, pvSizeByte, "g", pv
}

// create lvm volume
func (ns *nodeServer) createVolume(ctx context.Context, volumeID, vgName, pvType, lvmType string) error {
	pvSize, _, unit, _ := ns.getPvInfo(volumeID)

	pvNumber := 0
	var err error
	// Create VG if vg not exist,
	if pvType == LocalDisk {
		if pvNumber, err = createVG(vgName); err != nil {
			return err
		}
	}

	// check vg exist
	ckCmd := fmt.Sprintf("%s vgck %s", NsenterCmd, vgName)
	_, err = utils.Run(ckCmd)
	if err != nil {
		log.Errorf("createVolume:: VG is not exist: %s", vgName)
		return err
	}

	// Create lvm volume
	if lvmType == StripingType {
		cmd := fmt.Sprintf("%s lvcreate -i %d -n %s -L %d%s %s", NsenterCmd, pvNumber, volumeID, pvSize, unit, vgName)
		_, err = utils.Run(cmd)
		if err != nil {
			return err
		}
		log.Infof("Successful Create Striping LVM volume: %s, Size: %d%s, vgName: %s, striped number: %d", volumeID, pvSize, unit, vgName, pvNumber)
	} else if lvmType == LinearType {
		cmd := fmt.Sprintf("%s lvcreate -n %s -L %d%s %s", NsenterCmd, volumeID, pvSize, unit, vgName)
		_, err = utils.Run(cmd)
		if err != nil {
			return err
		}
		log.Infof("Successful Create Linear LVM volume: %s, Size: %d%s, vgName: %s", volumeID, pvSize, unit, vgName)
	}
	return nil
}
