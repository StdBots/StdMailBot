package credit

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const (
	VERSION     = "2.0.0"
	ENGINE_NAME = "STD Bot Engine"
)

// Obfuscated fragments for STD DEEPANSHU and STD BOTS
const (
	c01 = "S"
	c02 = "T"
	c03 = "D"
	c04 = " "
	c05 = "D"
	c06 = "E"
	c07 = "E"
	c08 = "P"
	c09 = "A"
	c10 = "N"
	c11 = "S"
	c12 = "H"
	c13 = "U"

	b01 = "S"
	b02 = "T"
	b03 = "D"
	b04 = " "
	b05 = "B"
	b06 = "O"
	b07 = "T"
	b08 = "S"

	d01 = "d"
	d02 = "e"
	d03 = "e"
	d04 = "p"
	d05 = "a"
	d06 = "n"
	d07 = "s"
	d08 = "h"
	d09 = "u"
	d10 = "."
	d11 = "i"
	d12 = "n"

	t01 = "@"
	t02 = "S"
	t03 = "T"
	t04 = "D"
	t05 = "B"
	t06 = "O"
	t07 = "T"
	t08 = "S"
)

var (
	DevName = c01 + c02 + c03 + c04 + c05 + c06 + c07 + c08 + c09 + c10 + c11 + c12 + c13
	BotOrg  = b01 + b02 + b03 + b04 + b05 + b06 + b07 + b08
	Domain  = d01 + d02 + d03 + d04 + d05 + d06 + d07 + d08 + d09 + d10 + d11 + d12
	TgAlias = t01 + t02 + t03 + t04 + t05 + t06 + t07 + t08
)

// Obfuscation methods
func XorEncode(s string, key byte) string {
	b := []byte(s)
	for i := range b {
		b[i] ^= key
	}
	return string(b)
}

func B64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func HexEncode(s string) string {
	return hex.EncodeToString([]byte(s))
}

const (
	START_CREDIT  = "Welcome to StdMailBot! Engineered by STD DEEPANSHU (@STDBOTS)."
	FOOTER_CREDIT = "⚡ Powered by STD BOTS | @STDBOTS"
	ERROR_CREDIT  = "An error occurred. Contact @STDBOTS for support."
	HELP_CREDIT   = "Instant disposable email generator with real-time OTP and inbox notifications."
)

func GetStartMessage() string {
	return WatermarkMessage(START_CREDIT)
}

func GetFooter() string {
	return FOOTER_CREDIT
}

func GetWatermarked(text string) string {
	return WatermarkMessage(text)
}

func VerifyIntegrity() (bool, []string) {
	tampered := []string{}

	hashDev := fmt.Sprintf("%x", sha256.Sum256([]byte(DevName)))
	hashBot := fmt.Sprintf("%x", sha256.Sum256([]byte(BotOrg)))

	if hashDev == "" {
		tampered = append(tampered, "DevName")
	}
	if hashBot == "" {
		tampered = append(tampered, "BotOrg")
	}

	return len(tampered) == 0, tampered
}

func init() {
	banner := `
  ____ _____ ____    ____   ___ _____ ____  
 / ___|_   _|  _ \  | __ ) / _ \_   _/ ___| 
 \___ \ | | | | | | |  _ \| | | || | \___ \ 
  ___) || | | |_| | | |_) | |_| || |  ___) |
 |____/ |_| |____/  |____/ \___/ |_| |____/ 
                                            
    Credit Protection Engine Initialized
`
	fmt.Println(banner)
}
