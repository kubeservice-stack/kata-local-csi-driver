package csi

import (
	"os"
	"strings"

	"github.com/kubeservice-stack/kata-local-csi-driver/pkg/utils"
	mountutils "k8s.io/mount-utils"
	utilexec "k8s.io/utils/exec"
)

type ExecRunner interface {
	Remove(name string) error
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	RunCommand(cmd string) (string, error)
	IsBlockDevice(fullPath string) (bool, error)
	MountBlock(source, target string, opts ...string) error
	EnsureBlock(target string) error
	CleanupMountPoint(mountPath string, mounter mountutils.Interface, extensiveMountPointCheck bool) error
	ResizeFS(devicePath string, deviceMountPath string) (bool, error)
}

type execRunner struct{}

func NewExecRunner() ExecRunner {
	return &execRunner{}
}

func (tool *execRunner) Remove(name string) error {
	return os.Remove(name)
}

func (tool *execRunner) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (tool *execRunner) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (tool *execRunner) IsBlockDevice(fullPath string) (bool, error) {
	return utils.IsBlockDevice(fullPath)
}

func (tool *execRunner) RunCommand(cmd string) (string, error) {
	return utils.Run(cmd)
}

func (tool *execRunner) MountBlock(source, target string, opts ...string) error {
	return utils.MountBlock(source, target, opts...)
}

func (tool *execRunner) EnsureBlock(target string) error {
	return utils.EnsureBlock(target)
}

func (tool *execRunner) CleanupMountPoint(mountPath string, mounter mountutils.Interface, extensiveMountPointCheck bool) error {
	return mountutils.CleanupMountPoint(mountPath, mounter, extensiveMountPointCheck)
}

func (tool *execRunner) ResizeFS(devicePath string, deviceMountPath string) (bool, error) {
	resizer := mountutils.NewResizeFs(utilexec.New())
	return resizer.Resize(devicePath, deviceMountPath)
}

type fakeExecRunner struct{}

func NewFakeExecRunner() ExecRunner {
	return &fakeExecRunner{}
}

func (tool *fakeExecRunner) Remove(name string) error {
	return nil
}

func (tool *fakeExecRunner) Stat(name string) (os.FileInfo, error) {
	if strings.Contains(name, "snapshot") {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func (tool *fakeExecRunner) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (tool *fakeExecRunner) RunCommand(cmd string) (string, error) {
	return "", nil
}

func (tool *fakeExecRunner) IsBlockDevice(fullPath string) (bool, error) {
	return true, nil
}

func (tool *fakeExecRunner) MountBlock(source, target string, opts ...string) error {
	return nil
}

func (tool *fakeExecRunner) EnsureBlock(target string) error {
	return nil
}

func (tool *fakeExecRunner) CleanupMountPoint(mountPath string, mounter mountutils.Interface, extensiveMountPointCheck bool) error {
	return nil
}

func (tool *fakeExecRunner) ResizeFS(devicePath string, deviceMountPath string) (bool, error) {
	return true, nil
}
