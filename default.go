package sirdataConf

import "time"

func SetDefaultString(entry *string, def string) {
	if *entry == "" {
		*entry = def
	}
}

func SetDefaultInt(entry *int, def int) {
	if *entry == 0 {
		*entry = def
	}
}

func SetDefaultUInt16(entry *uint16, def uint16) {
	if *entry == 0 {
		*entry = def
	}
}

func SetDefaultUint(entry *uint, def uint) {
	if *entry == 0 {
		*entry = def
	}
}

func SetDefaultFloat(entry *float64, def float64) {
	if *entry == 0 {
		*entry = def
	}
}

func SetDefaultDuration(entry *time.Duration, def time.Duration) {
	if *entry == time.Duration(0) {
		*entry = def
	}
}
