// Package sdwire provides a Go SDK for controlling SDWireC devices.
// SDWireC is a USB-controlled SD card multiplexer that allows switching
// an SD card between a Device Under Test (DUT) and Test System (TS).
package sdwire

import (
	"fmt"

	"github.com/google/gousb"
)

const (
	SDWireCVID         = 0x04E8
	SDWireCPID         = 0x6001
	SDWireCProductName = "sd-wire"

	SDWire3VID = 0x0BDA
	SDWire3PID = 0x0316
)

// DeviceGeneration represents the generation/type of SDWire device.
type DeviceGeneration int

const (
	// GenerationSDWireC represents the original SDWireC device using FTDI control.
	GenerationSDWireC DeviceGeneration = iota
	// GenerationSDWire3 represents the SDWire3 device using kernel driver attach/detach.
	GenerationSDWire3
)

// String returns a human-readable description of the device generation.
func (g DeviceGeneration) String() string {
	switch g {
	case GenerationSDWireC:
		return "SDWireC"
	case GenerationSDWire3:
		return "SDWire3"
	default:
		return "Unknown"
	}
}

// SwitchMode represents the SD card connection mode.
type SwitchMode int

const (
	// ModeTarget connects the SD card to the target device being tested.
	ModeTarget SwitchMode = iota
	// ModeHost connects the SD card to the host computer for flashing/access.
	ModeHost
)

// String returns a human-readable description of the switch mode.
func (m SwitchMode) String() string {
	switch m {
	case ModeTarget:
		return "Target"
	case ModeHost:
		return "Host"
	default:
		return "Unknown"
	}
}

const (
	ftdiSioSetBitmodeRequest = 0x0B
	ftdiSioBitmodeCbus       = 0x20
)

// DeviceController defines the interface for controlling different SDWire device generations.
type DeviceController interface {
	SetMode(mode SwitchMode) error
}

// SDWire represents a connected SDWire device that can switch an SD card
// between a target device and host computer.
type SDWire struct {
	device       *gousb.Device
	serial       string
	product      string
	manufacturer string
	generation   DeviceGeneration
	controller   DeviceController
}

// DeviceInfo contains identifying information about an SDWire device.
type DeviceInfo struct {
	Serial       string
	Product      string
	Manufacturer string
	Generation   DeviceGeneration
}

// ListDevices discovers all connected SDWire devices and returns their information.
// This is useful for device enumeration before connecting to a specific device.
func ListDevices() ([]*DeviceInfo, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	var devices []*DeviceInfo

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return (desc.Vendor == SDWireCVID && desc.Product == SDWireCPID) ||
			(desc.Vendor == SDWire3VID && desc.Product == SDWire3PID)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find USB devices: %w", err)
	}
	defer func() {
		for _, dev := range devs {
			dev.Close()
		}
	}()

	for _, dev := range devs {
		serial, err := dev.SerialNumber()
		if err != nil {
			serial = "unknown"
		}

		product, err := dev.Product()
		if err != nil {
			product = "unknown"
		}

		manufacturer, err := dev.Manufacturer()
		if err != nil {
			manufacturer = "unknown"
		}

		// Determine generation based on VID/PID
		desc := dev.Desc
		generation := GenerationSDWireC // Default to SDWireC
		if desc.Vendor == SDWire3VID && desc.Product == SDWire3PID {
			generation = GenerationSDWire3
		}

		devices = append(devices, &DeviceInfo{
			Serial:       serial,
			Product:      product,
			Manufacturer: manufacturer,
			Generation:   generation,
		})
	}

	return devices, nil
}

// New connects to the first available SDWire device.
// This is a convenience function for single-device setups.
// The returned SDWire must be closed with Close() when done.
func New() (*SDWire, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no SDWire devices found")
	}
	return NewWithSerial(devices[0].Serial)
}

// NewWithSerial connects to a specific SDWire device by its serial number.
// Use ListDevices() first to discover available devices and their serial numbers.
// The returned SDWire must be closed with Close() when done.
func NewWithSerial(serial string) (*SDWire, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return (desc.Vendor == SDWireCVID && desc.Product == SDWireCPID) ||
			(desc.Vendor == SDWire3VID && desc.Product == SDWire3PID)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find USB devices: %w", err)
	}

	for _, dev := range devs {
		deviceSerial, err := dev.SerialNumber()
		if err != nil {
			dev.Close()
			continue
		}

		if deviceSerial == serial {
			product, _ := dev.Product()
			manufacturer, _ := dev.Manufacturer()

			// Determine generation based on VID/PID
			desc := dev.Desc
			generation := GenerationSDWireC // Default to SDWireC
			if desc.Vendor == SDWire3VID && desc.Product == SDWire3PID {
				generation = GenerationSDWire3
			}

			// Create appropriate controller based on generation
			var controller DeviceController
			switch generation {
			case GenerationSDWireC:
				controller = &sdwireCController{device: dev}
			case GenerationSDWire3:
				controller = &sdwire3Controller{device: dev}
			default:
				dev.Close()
				return nil, fmt.Errorf("unsupported device generation: %v", generation)
			}

			return &SDWire{
				device:       dev,
				serial:       deviceSerial,
				product:      product,
				manufacturer: manufacturer,
				generation:   generation,
				controller:   controller,
			}, nil
		}
		dev.Close()
	}

	return nil, fmt.Errorf("SDWire device with serial %s not found", serial)
}

// Close releases the USB device connection. Always call this when done with the device.
func (s *SDWire) Close() error {
	if s.device != nil {
		return s.device.Close()
	}
	return nil
}

// GetSerial returns the device's USB serial number.
func (s *SDWire) GetSerial() string {
	return s.serial
}

// GetProduct returns the device's USB product name.
func (s *SDWire) GetProduct() string {
	return s.product
}

// GetManufacturer returns the device's USB manufacturer name.
func (s *SDWire) GetManufacturer() string {
	return s.manufacturer
}

// String returns a formatted string with device information.
func (s *SDWire) String() string {
	return fmt.Sprintf("%s\t[%s::%s]", s.serial, s.product, s.manufacturer)
}

// SetMode switches the SD card to the specified mode.
func (s *SDWire) SetMode(mode SwitchMode) error {
	return s.controller.SetMode(mode)
}


// sdwireCController implements DeviceController for SDWireC devices using FTDI control.
type sdwireCController struct {
	device *gousb.Device
}

// SetMode switches the SD card using FTDI bitmode control.
func (c *sdwireCController) SetMode(mode SwitchMode) error {
	if c.device == nil {
		return fmt.Errorf("device not initialized")
	}

	var target byte
	switch mode {
	case ModeTarget:
		target = 0
	case ModeHost:
		target = 1
	default:
		return fmt.Errorf("invalid switch mode: %v", mode)
	}

	// The Python code uses: ftdi.set_bitmode(0xF0 | target, Ftdi.BitMode.CBUS)
	// In FTDI terms: wValue = (mode << 8) | mask
	// where mode = FTDI_SIO_BITMODE_CBUS (0x20) and mask = 0xF0 | target
	value := uint16(ftdiSioBitmodeCbus<<8) | uint16(0xF0|target)

	_, err := c.device.Control(
		gousb.ControlOut|gousb.ControlVendor|gousb.ControlDevice,
		ftdiSioSetBitmodeRequest,
		value,
		0,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to set SDWire mode: %w", err)
	}

	return nil
}

// sdwire3Controller implements DeviceController for SDWire3 devices using kernel driver attach/detach.
type sdwire3Controller struct {
	device *gousb.Device
}

// SetMode switches the SD card using kernel driver attach/detach mechanism.
func (c *sdwire3Controller) SetMode(mode SwitchMode) error {
	if c.device == nil {
		return fmt.Errorf("device not initialized")
	}

	// Enable auto-detach so we can control kernel driver attachment
	err := c.device.SetAutoDetach(true)
	if err != nil {
		return fmt.Errorf("failed to enable auto-detach: %w", err)
	}

	switch mode {
	case ModeHost:
		// Switch to TS mode: ensure kernel driver is attached (don't claim interface)
		// Just reset the device - kernel driver should reattach automatically
		return c.device.Reset()

	case ModeTarget:
		// Switch to DUT mode: detach kernel driver by claiming interface 0, then reset
		cfg, err := c.device.Config(1)
		if err != nil {
			// If we can't get config, just reset - might work anyway
			return c.device.Reset()
		}
		defer cfg.Close()

		// Claim interface 0 to detach kernel driver
		intf, err := cfg.Interface(0, 0)
		if err == nil {
			// Successfully claimed interface (kernel driver detached)
			intf.Close() // Release interface but keep kernel driver detached
		}

		// Reset the device
		return c.device.Reset()

	default:
		return fmt.Errorf("invalid switch mode: %v", mode)
	}
}
