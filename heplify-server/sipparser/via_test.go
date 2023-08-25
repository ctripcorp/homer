// Copyright 2011, Shelby Ramsey. All rights reserved.
// Copyright 2018, Eugen Biegler. All rights reserved.
// Use of this code is governed by a BSD license that can be
// found in the LICENSE.txt file.

package sipparser

// Imports from the go standard library
import (
	"testing"
)

func TestVia(t *testing.T) {
	sm := &SipMsg{}
	s := "SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bK05B1a4c756d527cb513"
	sm.parseVia(s)
	if sm.Error != nil {
		t.Errorf("[TestVia] Error parsing via.  Received: %v", sm.Error)
	}
	/*
		if sm.Via[0].Proto != "SIP" {
			t.Errorf("[TestVia] Error parsing via \"SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bK05B1a4c756d527cb513\".  sm.Via[0].Proto should be \"SIP\" but received: \"%s\"", sm.Via[0].Proto)
		}
		if sm.Via[0].Version != "2.0" {
			t.Errorf("[TestVia] Error parsing via \"SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bK05B1a4c756d527cb513\".  sm.Via[0].Version should be \"2.0\" but received: \"%s\"", sm.Via[0].Version)
		}
		if sm.Via[0].Transport != "UDP" {
			t.Errorf("[TestVia] Error parsing via \"SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bK05B1a4c756d527cb513\".  sm.Via[0].Transport should be \"UDP\" but received: \"%s\"", sm.Via[0].Transport)
		}
		if sm.Via[0].SentBy != "0.0.0.0:5060" {
			t.Errorf("[TestVia] Error parsing via \"SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bK05B1a4c756d527cb513\".  Sent by should be \"0.0.0.0:5060\" but received: \"%s\".", sm.Via[0].SentBy)
		}
	*/
	if sm.ViaOne != s {
		t.Errorf("[TestVia] Error parsing via.  Received: %v", sm.Error)
	}
	if sm.ViaOneBranch != "z9hG4bK05B1a4c756d527cb513" {
		t.Errorf("[TestMultipleVias] Error parsing via %q. sm.ViaOneBranch should be \"z9hG4bK05B1a4c756d527cb513\" but received %q", s, sm.ViaOneBranch)
	}
}

func TestMultipleVias(t *testing.T) {
	sm := &SipMsg{}
	//s := "SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bKea28eb32f60dc,SIP/2.0/UDP 1.1.1.1:5060;branch=z9hG4bK1750901461"
	s := "SIP/2.0/UDP 0.0.0.0:5060;branch=z9hG4bKea28eb32f60dc;rport=5080"
	sm.parseVia(s)
	if sm.ViaOneBranch != "z9hG4bKea28eb32f60dc" {
		t.Errorf("[TestMultipleVias] Error parsing via %q. sm.ViaOneBranch should be \"z9hG4bKea28eb32f60dc\" but received %q", s, sm.ViaOneBranch)
	}
	/* 	if len(sm.Via) != 2 {
	   		t.Fatalf("[TestMultipleVias] Error parsing via %q. len(sm.Via) should be 2 but received %d", s, len(sm.Via))
	   	}
	   	if sm.Via[0].Branch != "z9hG4bKea28eb32f60dc" {
	   		t.Errorf("[TestMultipleVias] Error parsing via %q. sm.Via[0].Branch should be \"z9hG4bKea28eb32f60dc\" but received %q", s, sm.Via[0].Branch)
	   	}
	   	if sm.Via[1].Branch != "z9hG4bK1750901461" {
	   		t.Errorf("[TestMultipleVias] Error parsing via %q. sm.Via[1].Branch should be \"z9hG4bK1750901461\" but received %q", s, sm.Via[1].Branch)
	   	} */
}
