//go:build darwin && !no_vz
// +build darwin,!no_vz

package vz

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/lima-vm/lima/pkg/reflectutil"

	"github.com/Code-Hex/vz/v3"

	"github.com/lima-vm/lima/pkg/store/filenames"
	"github.com/sirupsen/logrus"

	"github.com/lima-vm/lima/pkg/driver"
	"github.com/lima-vm/lima/pkg/limayaml"
)

type LimaVzDriver struct {
	*driver.BaseDriver

	machine *vz.VirtualMachine
}

func New(driver *driver.BaseDriver) *LimaVzDriver {
	return &LimaVzDriver{
		BaseDriver: driver,
	}
}

func (l *LimaVzDriver) Validate() error {
	if *l.Yaml.MountType == limayaml.NINEP {
		return fmt.Errorf("field `mountType` must be %q or %q for VZ driver , got %q", limayaml.REVSSHFS, limayaml.VIRTIOFS, *l.Yaml.MountType)
	}
	if *l.Yaml.Firmware.LegacyBIOS {
		return fmt.Errorf("`firmware.legacyBIOS` configuration is not supported for VZ driver")
	}
	if unknown := reflectutil.UnknownNonEmptyFields(l.Yaml, "VMType",
		"Arch",
		"Images",
		"CPUs",
		"Memory",
		"Disk",
		"Mounts",
		"MountType",
		"SSH",
		"Firmware",
		"Provision",
		"Containerd",
		"Probes",
		"PortForwards",
		"Message",
		"Networks",
		"Env",
		"DNS",
		"HostResolver",
		"PropagateProxyEnv",
		"CACertificates",
		"Rosetta",
	); len(unknown) > 0 {
		logrus.Warnf("Ignoring: vmType %s: %+v", *l.Yaml.VMType, unknown)
	}

	for i, image := range l.Yaml.Images {
		if unknown := reflectutil.UnknownNonEmptyFields(image, "File"); len(unknown) > 0 {
			logrus.Warnf("Ignoring: vmType %s: images[%d]: %+v", *l.Yaml.VMType, i, unknown)
		}
	}

	for i, mount := range l.Yaml.Mounts {
		if unknown := reflectutil.UnknownNonEmptyFields(mount, "Location",
			"MountPoint",
			"Writable",
			"SSHFS",
			"NineP",
		); len(unknown) > 0 {
			logrus.Warnf("Ignoring: vmType %s: mounts[%d]: %+v", *l.Yaml.VMType, i, unknown)
		}
	}

	for i, network := range l.Yaml.Networks {
		if unknown := reflectutil.UnknownNonEmptyFields(network, "VZNAT",
			"MACAddress",
			"Interface",
		); len(unknown) > 0 {
			logrus.Warnf("Ignoring: vmType %s: networks[%d]: %+v", *l.Yaml.VMType, i, unknown)
		}
	}
	return nil
}

func (l *LimaVzDriver) CreateDisk() error {
	if err := EnsureDisk(l.BaseDriver); err != nil {
		return err
	}

	return nil
}

func (l *LimaVzDriver) Start(ctx context.Context) (chan error, error) {
	logrus.Infof("Starting VZ (hint: to watch the boot progress, see %q)", filepath.Join(l.Instance.Dir, filenames.SerialLog))
	vm, errCh, err := startVM(ctx, l.BaseDriver)
	if err != nil {
		if errors.Is(err, vz.ErrUnsupportedOSVersion) {
			panic(err)
			return nil, fmt.Errorf("vz driver requires macOS 13 or higher to run: %q", err)
		}
		return nil, err
	}
	l.machine = vm

	return errCh, nil
}

func (l *LimaVzDriver) Stop(_ context.Context) error {
	logrus.Info("Shutting down VZ")
	canStop := l.machine.CanStop()
	if canStop {
		return l.machine.Stop()
	}

	return nil
}
