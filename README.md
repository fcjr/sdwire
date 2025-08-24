# SDWire Go Library

[![Go Reference](https://pkg.go.dev/badge/github.com/fcjr/sdwire.svg)](https://pkg.go.dev/github.com/fcjr/sdwire)
[![Go Report Card](https://goreportcard.com/badge/github.com/fcjr/sdwire)](https://goreportcard.com/report/github.com/fcjr/sdwire)

Go library for controlling SDWireC devices - USB-controlled SD card multiplexers for automated testing and development.

## Features

🔀 **SD Card Switching** - Switch SD cards between target device and host computer  
🔍 **Device Discovery** - Automatically find and enumerate connected devices  
💾 **Device Information** - Access device serial, product, and manufacturer details  

## Installation

```bash
go get github.com/fcjr/sdwire
```

## Quick Start

```go
package main

import (
    "log"
    "github.com/fcjr/sdwire"
)

func main() {
    // Connect to the first available SDWireC device
    device, err := sdwire.New()
    if err != nil {
        log.Fatal(err)
    }
    defer device.Close()

    // Switch SD card to host computer for flashing
    device.SetMode(sdwire.ModeHost)
    
    // Flash your image here...
    
    // Switch SD card to target device for testing
    device.SetMode(sdwire.ModeTarget)
}
```

## Usage Examples

### Basic SD Card Switching

```go
device, err := sdwire.New()
if err != nil {
    log.Fatal(err)
}
defer device.Close()

// Switch to Host (for flashing/accessing SD card)
err = device.SetMode(sdwire.ModeHost)
if err != nil {
    log.Fatal(err)
}

// Switch to Target (for testing)
err = device.SetMode(sdwire.ModeTarget)
if err != nil {
    log.Fatal(err)
}
```

### Getting Device Information

```go
import (
    "fmt"
    "log"
    "github.com/fcjr/sdwire"
)

device, err := sdwire.New()
if err != nil {
    log.Fatal(err)
}
defer device.Close()

// Get device information
fmt.Printf("Device: %s [%s::%s]\n", 
    device.GetSerial(), 
    device.GetProduct(), 
    device.GetManufacturer())
```

### Managing Multiple Devices

```go
import (
    "fmt"
    "log"
    "github.com/fcjr/sdwire"
)

// List all connected SDWireC devices
devices, err := sdwire.ListDevices()
if err != nil {
    log.Fatal(err)
}

for i, info := range devices {
    fmt.Printf("Device %d: %s [%s::%s]\n", 
        i+1, info.Serial, info.Product, info.Manufacturer)
}

// Connect to a specific device by serial number
device, err := sdwire.NewWithSerial(devices[0].Serial)
if err != nil {
    log.Fatal(err)
}
defer device.Close()
```

## API Reference

### Types

```go
type SDWireC struct {
    // Represents a connected SDWireC device
}

type SwitchMode int
const (
    ModeTarget SwitchMode = iota  // Target device mode
    ModeHost                      // Host computer mode
)

type DeviceInfo struct {
    Serial       string
    Product      string
    Manufacturer string
}
```

### Connection Management

| Function | Description |
|----------|-------------|
| `New() (*SDWireC, error)` | Connect to the first available device |
| `NewWithSerial(serial string) (*SDWireC, error)` | Connect to device by serial number |
| `ListDevices() ([]*DeviceInfo, error)` | List all connected devices |
| `Close() error` | Close device connection |

### Device Control

| Function | Description |
|----------|-------------|
| `SetMode(mode SwitchMode) error` | Switch to specified mode |

### Device Information

| Function | Description |
|----------|-------------|
| `GetSerial() string` | Get device serial number |
| `GetProduct() string` | Get device product name |
| `GetManufacturer() string` | Get device manufacturer |

### Constants

```go
const (
    ModeTarget  // Connects SD card to target device
    ModeHost    // Connects SD card to host computer
)
```

## Hardware Requirements

- SDWireC device (USB-controlled SD card multiplexer)
- USB connection to host computer
- Proper USB permissions (see Troubleshooting)

## Supported Operating Systems

- **Linux** ✅ (Tested on Ubuntu, Debian)
- **macOS** ✅ (Tested on macOS 10.15+)
- **Windows** ✅ (Tested on Windows 10+)

## Troubleshooting

### Device Not Found

If `ListDevices()` returns no devices:

1. **Check USB connection** - Ensure the SDWireC is properly connected
2. **Check VID/PID** - SDWireC should appear as `04E8:6001` when listing USB devices
3. **Verify permissions** - Ensure you have proper USB device access permissions

### Permission Issues (Linux)

If you get permission errors on Linux:

1. **Add udev rules** - Create `/etc/udev/rules.d/99-sdwire.rules`:
   ```
   SUBSYSTEM=="usb", ATTR{idVendor}=="04e8", ATTR{idProduct}=="6001", MODE="0666"
   ```

2. **Reload udev rules**:
   ```bash
   sudo udevadm control --reload-rules
   sudo udevadm trigger
   ```

3. **Add user to plugdev group**:
   ```bash
   sudo usermod -a -G plugdev $USER
   ```

### Multiple Devices

When using multiple SDWireC devices:

```go
devices, err := sdwire.ListDevices()
if err != nil {
    log.Fatal(err)
}

for _, info := range devices {
    device, err := sdwire.NewWithSerial(info.Serial)
    if err != nil {
        continue
    }
    defer device.Close()
    
    // Use device...
    device.SetMode(sdwire.ModeTarget)
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Related Projects

- [sdwire-cli](https://github.com/Badger-Embedded/sdwire-cli) - Python CLI tool for SDWireC
- [ykush3](https://github.com/fcjr/ykush3) - Go library for YKUSH3 USB switches

---

**Made with ❤️ at the [@recursecenter](https://www.recurse.com/)**
