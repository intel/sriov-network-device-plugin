// Copyright 2018 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resources

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	registerapi "k8s.io/kubernetes/pkg/kubelet/apis/pluginregistration/v1"
)

type resourceServer struct {
	resourcePool       types.ResourcePool
	endPoint           string // Socket file
	resourceNamePrefix string
	grpcServer         *grpc.Server
	termSignal         chan bool
	updateSignal       chan bool
	stopWatcher        chan bool
	restartSignal      chan bool
	checkIntervals     int // health check intervals in seconds
}

func newResourceServer(prefix, suffix string, rp types.ResourcePool) types.ResourceServer {
	sockName := fmt.Sprintf("%s.%s", rp.GetResourceName(), suffix)
	return &resourceServer{
		resourcePool:       rp,
		endPoint:           sockName,
		resourceNamePrefix: prefix,
		grpcServer:         grpc.NewServer(),
		termSignal:         make(chan bool, 1),
		updateSignal:       make(chan bool),
		stopWatcher:        make(chan bool),
		restartSignal:      make(chan bool),
		checkIntervals:     20, // updates every 20 seconds
	}
}

func (rs *resourceServer) GetInfo(ctx context.Context, rqt *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	pluginInfoResponse := &registerapi.PluginInfo{
		Type:              registerapi.DevicePlugin,
		Name:              fmt.Sprintf("%s/%s", rs.resourceNamePrefix, rs.resourcePool.GetResourceName()),
		Endpoint:          filepath.Join(types.SockDir, rs.endPoint),
		SupportedVersions: []string{"v1alpha1", "v1beta1", "v1"},
	}
	return pluginInfoResponse, nil
}

func (rs *resourceServer) NotifyRegistrationStatus(ctx context.Context, regstat *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	if regstat.PluginRegistered {
		glog.Infof("Plugin: %s gets registered successfully at Kubelet\n", rs.endPoint)
	} else {
		glog.Infof("Plugin: %s failed to be registered at Kubelet: %v; restarting.\n", rs.endPoint, regstat.Error)
		rs.restartSignal <- true
	}
	return &registerapi.RegistrationStatusResponse{}, nil
}

func (rs *resourceServer) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	glog.Infof("Allocate() called with %+v", rqt)
	resp := new(pluginapi.AllocateResponse)
	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)
		containerResp.Devices = rs.resourcePool.GetDeviceSpecs(rs.resourcePool.GetDeviceFiles(), container.DevicesIDs)
		containerResp.Envs = rs.getEnvs(container.DevicesIDs)
		containerResp.Mounts = rs.resourcePool.GetMounts()
		resp.ContainerResponses = append(resp.ContainerResponses, containerResp)
	}
	glog.Infof("AllocateResponse send: %+v", resp)
	return resp, nil
}

func (rs *resourceServer) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {

	methodID := fmt.Sprintf("ListAndWatch(%s)", rs.resourcePool.GetResourceName()) // for logging purpose
	glog.Infof("%s invoked", methodID)
	// Send initial list of devices
	devs := make([]*pluginapi.Device, 0)
	resp := new(pluginapi.ListAndWatchResponse)
	for _, dev := range rs.resourcePool.GetDevices() {
		devs = append(devs, dev)
	}
	resp.Devices = devs
	glog.Infof("%s: send devices %v\n", methodID, resp)

	if err := stream.Send(resp); err != nil {
		glog.Errorf("%s: error: cannot update device states: %v\n", methodID, err)
		rs.grpcServer.Stop()
		return err
	}

	// listen for events: if updateSignal send new list of devices
	for {
		select {
		case <-rs.termSignal:
			// Terminate signal received; return from mehtod call
			glog.Infof("%s: terminate signal received", methodID)
			return nil
		case <-rs.updateSignal:
			// Device health changed; so send new device list
			glog.Infof("%s: device health changed!\n", methodID)
			newDevs := make([]*pluginapi.Device, 0)
			for _, dev := range rs.resourcePool.GetDevices() {
				newDevs = append(newDevs, dev)
			}
			resp.Devices = newDevs
			glog.Infof("%s: send updated devices %v", methodID, resp)

			if err := stream.Send(resp); err != nil {
				glog.Errorf("%s: error: cannot update device states: %v\n", methodID, err)
				return err
			}

		}
	}
}

func (rs *resourceServer) PreStartContainer(ctx context.Context, psRqt *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (rs *resourceServer) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

func (rs *resourceServer) Init() error {
	resourceName := rs.resourcePool.GetResourceName()
	glog.Infof("initializing %s device pool", resourceName)
	if err := rs.resourcePool.DiscoverDevices(); err != nil {
		return err
	}
	return nil
}

// gRPC server related
func (rs *resourceServer) Start() error {
	resourceName := rs.resourcePool.GetResourceName()
	_ = rs.cleanUp() // try tp clean up and continue
	glog.Infof("starting %s device plugin endpoint at: %s\n", resourceName, rs.endPoint)
	sockPath := filepath.Join(types.SockDir, rs.endPoint)
	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		glog.Errorf("error starting %s device plugin endpoint: %v", resourceName, err)
		return err
	}

	// Register all services
	registerapi.RegisterRegistrationServer(rs.grpcServer, rs)
	pluginapi.RegisterDevicePluginServer(rs.grpcServer, rs)

	go rs.grpcServer.Serve(lis)
	// Wait for server to start by launching a blocking connection
	conn, err := grpc.Dial(sockPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		glog.Errorf("error. unable to establish test connection with %s gRPC server: %v", resourceName, err)
		return err
	}
	glog.Infof("%s device plugin endpoint started serving", resourceName)
	conn.Close()

	rs.triggerUpdate()
	return nil
}

func (rs *resourceServer) restart() error {
	resourceName := rs.resourcePool.GetResourceName()
	glog.Infof("restarting %s device plugin server...", resourceName)
	if rs.grpcServer == nil {
		return fmt.Errorf("grpc server instance not found for %s", resourceName)
	}
	rs.grpcServer.Stop()
	rs.grpcServer = nil
	// Send terminate signal to ListAndWatch()
	rs.termSignal <- true

	rs.grpcServer = grpc.NewServer() // new instance of a grpc server
	return rs.Start()
}

func (rs *resourceServer) Stop() error {
	resourceName := rs.resourcePool.GetResourceName()
	glog.Infof("stopping %s device plugin server...", resourceName)
	if rs.grpcServer == nil {
		return nil
	}
	// Send terminate signal to ListAndWatch()
	rs.termSignal <- true
	rs.stopWatcher <- true

	rs.grpcServer.Stop()
	rs.grpcServer = nil

	return rs.cleanUp()
}

func (rs *resourceServer) Watch() {
	select {
	case stop := <-rs.stopWatcher:
		if stop {
			return
		}
	case restart := <-rs.restartSignal:
		if restart {
			time.Sleep(time.Second * time.Duration(5))
			if err := rs.restart(); err != nil {
				glog.Fatalf("unable to restart server %v", err)
			}
		}
	}
}

func (rs *resourceServer) cleanUp() error {
	sockPath := filepath.Join(types.SockDir, rs.endPoint)
	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (rs *resourceServer) triggerUpdate() {
	rp := rs.resourcePool
	if rs.checkIntervals > 0 {
		go func() {
			for {
				changed := rp.Probe(rs.resourcePool.GetConfig(), rp.GetDevices())
				if changed {
					rs.updateSignal <- true
				}
				time.Sleep(time.Second * time.Duration(rs.checkIntervals))
			}
		}()
	}
}

func (rs *resourceServer) getEnvs(deviceIDs []string) map[string]string {

	varPrefix := "PCIDEVICE"
	resourceNamePrefix := strings.Replace(rs.resourceNamePrefix, ".", "_", -1)

	envs := rs.resourcePool.GetEnvs(deviceIDs)

	newEnvs := make(map[string]string)
	for k, v := range envs {
		exportedVar := strings.ToUpper(fmt.Sprintf("%s_%s_%s", varPrefix, resourceNamePrefix, k))
		newEnvs[exportedVar] = v
	}
	return newEnvs
}