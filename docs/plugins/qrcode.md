# QR Code Plugin

Pure Go QR code generation plugin for URLs, OTP/TOTP setup, and general use.

## Installation

```bash
gospa add qrcode
```

## Usage

```go
import "github.com/aydenstechdungeon/gospa/plugin/qrcode"

// Generate a QR code as data URL
dataURL, _ := qrcode.GenerateDataURL("https://example.com")

// Generate for OTP/TOTP setup
qrDataURL, _ := qrcode.ForOTP(otpURL)
```
