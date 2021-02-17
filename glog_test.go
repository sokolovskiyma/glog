package glog

import (
	"bytes"
	"testing"
)

func TestInfo(t *testing.T) {
	var out bytes.Buffer
	config.info.SetOutput(&out)
	config.info.SetFlags(0)

	Info("Test")
	if out.String() != "INFO: [Test]\n" {
		t.Error("Info fail")
	}
}

// func TestTelegram(t *testing.T) {
// 	var out bytes.Buffer
// 	config.info.SetOutput(&out)
// 	config.info.SetFlags(0)

// 	Info("Test")
// 	if out.String() != "INFO: [Test]\n" {
// 		t.Error("Info fail")
// 	}
// }
