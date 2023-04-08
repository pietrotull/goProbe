/////////////////////////////////////////////////////////////////////////////////
//
// Attribute_test.go
//
// Written by Lorenz Breidenbach lob@open.ch, November 2015
// Copyright (c) 2015 Open Systems AG, Switzerland
// All Rights Reserved.
//
/////////////////////////////////////////////////////////////////////////////////

package types

import (
	"reflect"
	"testing"
)

var (
	Sip      = [16]byte{0xA1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	Dip      = [16]byte{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5, 8, 9, 7, 9, 3}
	Dport    = []byte{0xCB, 0xF1}
	Protocol = uint8(6)
	Time     = 0
)

var tests = []struct {
	Attribute        Attribute
	Name             string
	ExtractedStrings string
}{
	{SipAttribute{ipAttribute{data: Sip[:]}}, "sip", "a102:304:506:708:90a:b0c:d0e:f10"},
	{DipAttribute{ipAttribute{data: Dip[:]}}, "dip", "301:401:509:206:503:508:907:903"},
	{DportAttribute{Dport}, "dport", "52209"},
	{ProtoAttribute{Protocol}, "proto", "TCP"},
}

func TestAttributes(t *testing.T) {
	for _, test := range tests {
		if test.Attribute.Name() != test.Name {
			t.Fatalf("wrong name")
		}
		es := test.Attribute.String()
		if !reflect.DeepEqual(es, test.ExtractedStrings) {
			t.Fatalf("%s: expected: %s got: %s", test.Attribute.Name(), test.ExtractedStrings, es)
		}
	}
}

func TestNewAttribute(t *testing.T) {
	for _, name := range []string{"sip", "dip", "dport", "proto"} {
		attrib, err := NewAttribute(name)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
		if name != attrib.Name() {
			t.Fatalf("Wrong attribute")
		}
	}

	attrib, err := NewAttribute("src")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if "sip" != attrib.Name() {
		t.Fatalf("Wrong attribute")
	}

	attrib, err = NewAttribute("dst")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if "dip" != attrib.Name() {
		t.Fatalf("Wrong attribute")
	}

	_, err = NewAttribute("time")
	if err == nil {
		t.Fatalf("Expected error")
	}
}

var parseQueryTypeTests = []struct {
	InQueryType     string
	OutAttributes   []Attribute
	OutHasAttrTime  bool
	OutHasAttrIface bool
	Success         bool
}{
	{"sip", []Attribute{SipAttribute{}}, false, false, true},
	{"src", []Attribute{SipAttribute{}}, false, false, true},
	{"dst", []Attribute{DipAttribute{}}, false, false, true},
	{"talk_src", []Attribute{SipAttribute{}}, false, false, true},
	{"sip,dip,dip,sip,dport", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, false, false, true},
	{"sip,dip,dip,iface,sip,dport", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, false, true, true},
	{"sip,dip,dst,src,dport", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, false, false, true},
	{"src,dst,dip,sip,dport", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, false, false, true},
	{"sip,dip,dip,sip,dport,talk_src", nil, false, false, false},
	{"sip,dip,time,dip,sip,dport", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, true, false, true},
	{"talk_src,dip", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, false, false, false},
	{"talk_src,src", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}}, false, false, false},
	{"raw", []Attribute{SipAttribute{}, DipAttribute{}, DportAttribute{}, ProtoAttribute{}}, true, true, true},
}

func TestParseQueryType(t *testing.T) {
	for _, test := range parseQueryTypeTests {
		attributes, selector, err := ParseQueryType(test.InQueryType)
		if !test.Success {
			if err == nil {
				t.Fatalf("Expected to fail on input %v but it didn't. Instead it output %v, %v",
					test.InQueryType, attributes, selector.Timestamp)
			}
		} else {
			if err != nil {
				t.Fatalf("Unexpectedly failed on input %v. The error is: %s",
					test.InQueryType, err)
			}
			if !reflect.DeepEqual(test.OutAttributes, attributes) ||
				test.OutHasAttrTime != selector.Timestamp ||
				test.OutHasAttrIface != selector.Iface {
				t.Fatalf("Returned an unexpected output. Expected output: %v, %v, %v. Actual output: %v, %v, %v",
					test.OutAttributes, test.OutHasAttrTime, test.OutHasAttrIface,
					attributes, selector.Timestamp, selector.Iface,
				)
			}
		}
	}
}
