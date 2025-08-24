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
)

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

// SDWireC represents a connected SDWireC device that can switch an SD card
// between a target device and host computer.
type SDWireC struct {
	device       *gousb.Device
	serial       string
	product      string
	manufacturer string
}

// DeviceInfo contains identifying information about an SDWireC device.
type DeviceInfo struct {
	Serial       string
	Product      string
	Manufacturer string
}

// ListDevices discovers all connected SDWireC devices and returns their information.
// This is useful for device enumeration before connecting to a specific device.
func ListDevices() ([]*DeviceInfo, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	var devices []*DeviceInfo

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == SDWireCVID && desc.Product == SDWireCPID
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

		devices = append(devices, &DeviceInfo{
			Serial:       serial,
			Product:      product,
			Manufacturer: manufacturer,
		})
	}

	return devices, nil
}

// New connects to the first available SDWireC device.
// This is a convenience function for single-device setups.
// The returned SDWireC must be closed with Close() when done.
func New() (*SDWireC, error) {
	devices, err := ListDevices()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no SDWireC devices found")
	}
	return NewWithSerial(devices[0].Serial)
}

// NewWithSerial connects to a specific SDWireC device by its serial number.
// Use ListDevices() first to discover available devices and their serial numbers.
// The returned SDWireC must be closed with Close() when done.
func NewWithSerial(serial string) (*SDWireC, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	devs, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == SDWireCVID && desc.Product == SDWireCPID
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

			return &SDWireC{
				device:       dev,
				serial:       deviceSerial,
				product:      product,
				manufacturer: manufacturer,
			}, nil
		}
		dev.Close()
	}

	return nil, fmt.Errorf("SDWireC device with serial %s not found", serial)
}

// Close releases the USB device connection. Always call this when done with the device.
func (s *SDWireC) Close() error {
	if s.device != nil {
		return s.device.Close()
	}
	return nil
}

// GetSerial returns the device's USB serial number.
func (s *SDWireC) GetSerial() string {
	return s.serial
}

// GetProduct returns the device's USB product name.
func (s *SDWireC) GetProduct() string {
	return s.product
}

// GetManufacturer returns the device's USB manufacturer name.
func (s *SDWireC) GetManufacturer() string {
	return s.manufacturer
}

// String returns a formatted string with device information.
func (s *SDWireC) String() string {
	return fmt.Sprintf("%s\t[%s::%s]", s.serial, s.product, s.manufacturer)
}

// SetMode switches the SD card to the specified mode.
func (s *SDWireC) SetMode(mode SwitchMode) error {
	var target byte
	switch mode {
	case ModeTarget:
		target = 0
	case ModeHost:
		target = 1
	default:
		return fmt.Errorf("invalid switch mode: %v", mode)
	}
	
	return s.setSdwire(target)
}

// setSdwire controls the FTDI CBUS pins to switch the SD card multiplexer.
// target=0 switches to target device, target=1 switches to host computer.
// This uses the FTDI bitmode control with CBUS pins 4-7 set high (0xF0)
// and the target value in the lowest bit.
func (s *SDWireC) setSdwire(target byte) error {
	if s.device == nil {
		return fmt.Errorf("device not initialized")
	}

	// The Python code uses: ftdi.set_bitmode(0xF0 | target, Ftdi.BitMode.CBUS)
	// In FTDI terms: wValue = (mode << 8) | mask
	// where mode = FTDI_SIO_BITMODE_CBUS (0x20) and mask = 0xF0 | target
	value := uint16(ftdiSioBitmodeCbus<<8) | uint16(0xF0|target)
	
	_, err := s.device.Control(
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