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

package metrics

const (
	// Subsystem is prometheus subsystem name.
	Subsystem = "local"
)

var (
	// Allocatable size of VG.
	VolumeGroupTotal = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"vgname":   "",
	}).SubScope(Subsystem).Gauge("volume_group_total")

	//Used size of VG.
	VolumeGroupUsedByLocal = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"vgname":   "",
	}).SubScope(Subsystem).Gauge("volume_group_used")

	//Total size of MountPoint.
	MountPointTotal = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"name":     "",
		"type":     "",
	}).SubScope(Subsystem).Gauge("mount_point_total")

	//Available size of MountPoint.
	MountPointAvailable = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"name":     "",
		"type":     "",
	}).SubScope(Subsystem).Gauge("mount_point_available")

	//Is MountPoint Bind.
	MountPointBind = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"name":     "",
	}).SubScope(Subsystem).Gauge("mount_point_bind")

	//Total size of Device.
	DeviceTotal = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"name":     "",
		"type":     "",
	}).SubScope(Subsystem).Gauge("device_total")

	//Available size of Device.
	DeviceAvailable = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"name":     "",
		"type":     "",
	}).SubScope(Subsystem).Gauge("device_available")

	//Is Device Bind.
	DeviceBind = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
		"name":     "",
	}).SubScope(Subsystem).Gauge("device_bind")

	//allocated number.
	AllocatedNum = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename": "",
	}).SubScope(Subsystem).Gauge("allocated_num")

	// storage_name
	// LVM:         VG name
	// MountPoint:  mount path
	// Device:      device path
	LocalPV = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename":     "",
		"pv_name":      "",
		"pv_type":      "",
		"pvc_name":     "",
		"pvc_ns":       "",
		"status":       "",
		"storage_name": "",
	}).SubScope(Subsystem).Gauge("local_pv")

	//pod inline volume.
	InlineVolume = DefaultTallyScope.Scope.Tagged(map[string]string{
		"nodename":      "",
		"pod_namespace": "",
		"pod_name":      "",
		"vgname":        "",
		"volume_name":   "",
	}).SubScope(Subsystem).Gauge("inline_volume")
)
