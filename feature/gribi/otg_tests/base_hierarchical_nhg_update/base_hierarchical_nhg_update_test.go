// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package base_hierarchical_nhg_update_test

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/open-traffic-generator/snappi/gosnappi"
	"github.com/openconfig/featureprofiles/internal/attrs"
	"github.com/openconfig/featureprofiles/internal/deviations"
	"github.com/openconfig/featureprofiles/internal/fptest"
	"github.com/openconfig/featureprofiles/internal/gribi"
	"github.com/openconfig/featureprofiles/internal/otgutils"
	"github.com/openconfig/gribigo/chk"
	"github.com/openconfig/gribigo/client"
	"github.com/openconfig/gribigo/fluent"
	"github.com/openconfig/ondatra"
	"github.com/openconfig/ondatra/gnmi"
	"github.com/openconfig/ondatra/gnmi/oc"
	"github.com/openconfig/ondatra/netutil"
	"github.com/openconfig/ygnmi/ygnmi"
	"github.com/openconfig/ygot/ygot"
)

const (
	vrfName = "VRF-1"

	// Destination ATE MAC address for port-2 and port-3.
	pMAC = "00:1A:11:00:1A:BC"
	// 12-bit filter for egress flow tracking. ABC in hex == 0xabc in hexadecimal.
	pMACFilter = "0xabc"
	// 15-bit filter for egress flow tracking. 1ABC in hex == 43981 in decimal.
	// pMACFilter      = "6844"
	// pMACFilterport2 = "6841"
	// pMACFilterport3 = "6842"
	// pMACFilterport4 = "6843"

	// port-2 nexthop ID.
	p2NHID = 40
	// port-3 nexthop ID.
	p3NHID = 41

	// VirtualIP route next-hop-group ID.
	virtualIPNHGID = 42
	// VirtualIP route nexthop.
	virtualIP = "203.0.113.1"
	// VirtualIP route prefix.
	virtualPfx = "203.0.113.1/32"

	// Destination route next-hop ID.
	dstNHID = 43
	// Destination route next-hop-group ID.
	dstNHGID = 44
	// Destination route prefix for DUT to ATE traffic.
	dstPfx            = "198.51.100.0/24"
	dstPfxFlowIP      = "198.51.100.0"
	ipv4PrefixLen     = 30
	ipv4FlowCount     = 65000
	innerSrcIPv4Start = "198.18.0.0"
	innerDstIPv4Start = "198.19.0.0"

	// load balancing precision, %. Defines expected +-% delta for ECMP flows.
	// E.g. 48-52% with two equal-weighted NHs.
	lbPrecision     = 2
	nh1ID           = 1
	nh2ID           = 2
	nh3ID           = 3
	nh10ID          = 10
	nh20ID          = 20
	nhg1ID          = 1
	nhg10ID         = 10
	nhg20ID         = 20
	innerDst        = "198.18.0.0/18"
	innerDstPfx     = "198.18.0.0"
	dstPfxvrf1      = "198.18.64.0"
	dstPfxvalue     = "198.18.196.0"
	mask            = "32"
	port2mac        = "00:1A:11:00:1A:B9"
	port3mac        = "00:1A:11:00:1A:BA"
	port4mac        = "00:1A:11:00:1A:BB"
	vip1            = "198.18.196.1"
	outerSrcIP      = "203.0.113.0"
	fps             = 1000000 // traffic frames per second
	innerSrcIP      = "198.51.100.61"
	vrfPrefixcount  = 10000
	ipv4Prefixcount = 700
)

type testArgs struct {
	dut    *ondatra.DUTDevice
	ate    *ondatra.ATEDevice
	top    gosnappi.Config
	ctx    context.Context
	client *gribi.Client
}

var (
	dutPort1 = attrs.Attributes{
		Desc:    "dutPort1",
		IPv4:    "192.0.2.1",
		IPv4Len: 30,
	}
	dutPort2 = attrs.Attributes{
		Desc:    "dutPort2",
		IPv4:    "192.0.2.5",
		IPv4Len: 30,
	}
	dutPort3 = attrs.Attributes{
		Desc:    "dutPort3",
		IPv4:    "192.0.2.9",
		IPv4Len: 30,
	}
	dutPort4 = attrs.Attributes{
		Desc:    "dutPort4",
		IPv4:    "192.0.2.13",
		IPv4Len: 30,
	}
	atePort1 = attrs.Attributes{
		Name:    "atePort1",
		IPv4:    "192.0.2.2",
		IPv4Len: 30,
	}
	atePort2 = attrs.Attributes{
		Name:    "atePort2",
		IPv4:    "192.0.2.6",
		IPv4Len: 30,
	}
	atePort3 = attrs.Attributes{
		Name:    "atePort3",
		IPv4:    "192.0.2.10",
		IPv4Len: 30,
	}
	atePort4 = attrs.Attributes{
		Name:    "atePort4",
		IPv4:    "192.0.2.14",
		IPv4Len: 30,
	}
	dutPort2DummyIP = attrs.Attributes{
		Desc:    "dutPort2",
		IPv4:    "192.0.2.21",
		IPv4Len: 30,
	}
	dutPort3DummyIP = attrs.Attributes{
		Desc:    "dutPort3",
		IPv4:    "192.0.2.41",
		IPv4Len: 30,
	}
	atePort2DummyIP = attrs.Attributes{
		Desc:    "atePort2",
		IPv4:    "192.0.2.22",
		IPv4Len: 32,
	}
	atePort3DummyIP = attrs.Attributes{
		Desc:    "atePort3",
		IPv4:    "192.0.2.42",
		IPv4Len: 32,
	}
	btrunk2, btrunk3, btrunk4 string
)

type bundleName struct {
	trunk2 string
	trunk3 string
	trunk4 string
}

func TestMain(m *testing.M) {
	fptest.RunTests(m)
}

// TE3.7 - BaseHierarchical NHG Update Tests.
func TestBaseHierarchicalNHGUpdate(t *testing.T) {
	ctx := context.Background()

	dut := ondatra.DUT(t, "dut")
	configureDUT(t, dut)

	ate := ondatra.ATE(t, "ate")
	top := configureATE(t, ate)

	ate.OTG().PushConfig(t, top)
	ate.OTG().StartProtocols(t)

	tests := []struct {
		name string
		desc string
		fn   func(ctx context.Context, t *testing.T, args *testArgs)
	}{
		{
			name: "testBaseHierarchialNHG",
			desc: "Usecase for NHG update in hierarchical resolution scenario",
			fn:   testBaseHierarchialNHG,
		},
		{
			name: "testImplementDrain",
			desc: "Usecase for Implementing Drain test",
			fn:   testImplementDrain,
		},
	}
	// Configure the gRIBI client
	client := gribi.Client{
		DUT:         dut,
		FIBACK:      true,
		Persistence: true,
	}
	defer client.Close(t)
	defer client.FlushAll(t)
	if err := client.Start(t); err != nil {
		t.Fatalf("gRIBI Connection can not be established")
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Name: %s", tt.name)
			t.Logf("Description: %s", tt.desc)
			// Flush past entries before running the tc
			client.BecomeLeader(t)
			client.FlushAll(t)
			tcArgs := &testArgs{
				ctx:    ctx,
				client: &client,
				dut:    dut,
				ate:    ate,
				top:    top,
			}
			tt.fn(ctx, t, tcArgs)
		})
	}
}

// TE3.7 - case 1: testBaseHierarchialNHG.
func testBaseHierarchialNHG(ctx context.Context, t *testing.T, args *testArgs) {

	t.Log("Create flows for port 1 to port2, port 1 to port3")
	p2flow := "Port 1 to Port 2"
	p3flow := "Port 1 to Port 3"
	p2Flow := createFlow(t, p2flow, args.top, &atePort2)
	p3Flow := createFlow(t, p3flow, args.top, &atePort3)

	defer func() {

		t.Log("Unconfig interfaces")
		deleteinterfaceconfig(t, args.dut)
		if deviations.GRIBIMACOverrideStaticARPStaticRoute(args.dut) {
			sp := gnmi.OC().NetworkInstance(deviations.DefaultNetworkInstance(args.dut)).Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_STATIC, deviations.StaticProtocolName(args.dut))
			gnmi.Delete(t, args.dut, sp.Static(atePort2DummyIP.IPv4CIDR()).Config())
			gnmi.Delete(t, args.dut, sp.Static(atePort3DummyIP.IPv4CIDR()).Config())
		}
	}()

	dutP2 := args.dut.Port(t, "port2").Name()
	dutP3 := args.dut.Port(t, "port3").Name()

	t.Logf("Adding gribi routes and validating traffic forwarding via port %v and NH ID %v", dutP2, p2NHID)

	var nh fluent.GRIBIEntry
	var op1, op3 *client.OpResult
	if !deviations.ExplicitGRIBIUnderNetworkInstance(args.dut) {
		if deviations.GRIBIMACOverrideWithStaticARP(args.dut) || deviations.GRIBIMACOverrideStaticARPStaticRoute(args.dut) {
			nh, op1 = gribi.NHEntry(p2NHID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: dutP2, Mac: pMAC, Dest: atePort2DummyIP.IPv4})
		} else {
			nh, op1 = gribi.NHEntry(p2NHID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: dutP2, Mac: pMAC})
		}
		nhg, op2 := gribi.NHGEntry(virtualIPNHGID, map[uint64]uint64{p2NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddEntries(t, []fluent.GRIBIEntry{nh, nhg}, []*client.OpResult{op1, op2})
		args.client.AddIPv4(t, virtualPfx, virtualIPNHGID, deviations.DefaultNetworkInstance(args.dut), deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)

		nh, op1 = gribi.NHEntry(dstNHID, virtualIP, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		nhg, op2 = gribi.NHGEntry(dstNHGID, map[uint64]uint64{dstNHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddEntries(t, []fluent.GRIBIEntry{nh, nhg}, []*client.OpResult{op1, op2})
	} else {
		args.client.AddNH(t, p2NHID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: dutP2, Mac: pMAC})
		args.client.AddNHG(t, virtualIPNHGID, map[uint64]uint64{p2NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddIPv4(t, virtualPfx, virtualIPNHGID, deviations.DefaultNetworkInstance(args.dut), deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)

		args.client.AddNH(t, dstNHID, virtualIP, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddNHG(t, dstNHGID, map[uint64]uint64{dstNHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
	}
	args.client.AddIPv4(t, dstPfx, dstNHGID, vrfName, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)

	waitOTGARPEntry(t)

	otgutils.WaitForARP(t, args.ate.OTG(), args.top, "IPv4")
	validateTrafficFlows(t, args.ate, []gosnappi.Flow{p2Flow}, []gosnappi.Flow{p3Flow}, nil, pMACFilter)

	t.Logf("Adding a new NH via port %v with ID %v", dutP3, p3NHID)
	if !deviations.ExplicitGRIBIUnderNetworkInstance(args.dut) {
		if deviations.GRIBIMACOverrideWithStaticARP(args.dut) || deviations.GRIBIMACOverrideStaticARPStaticRoute(args.dut) {
			nh, op3 = gribi.NHEntry(p3NHID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: dutP3, Mac: pMAC, Dest: atePort3DummyIP.IPv4})

		} else {
			nh, op3 = gribi.NHEntry(p3NHID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: dutP3, Mac: pMAC})
		}
	} else {
		args.client.AddNH(t, p3NHID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: dutP3, Mac: pMAC})
	}

	t.Logf("Performing implicit in-place replace with two next-hops (NH IDs: %v and %v)", p2NHID, p3NHID)
	if !deviations.ExplicitGRIBIUnderNetworkInstance(args.dut) {
		nhg, op2 := gribi.NHGEntry(virtualIPNHGID, map[uint64]uint64{p2NHID: 1, p3NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddEntries(t, []fluent.GRIBIEntry{nh, nhg}, []*client.OpResult{op1, op3, op2})
	} else {
		args.client.AddNHG(t, virtualIPNHGID, map[uint64]uint64{p2NHID: 1, p3NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
	}
	validateTrafficFlows(t, args.ate, nil, nil, []gosnappi.Flow{p2Flow, p3Flow}, pMACFilter)

	t.Logf("Performing implicit in-place replace using the next-hop with ID %v", p3NHID)
	if !deviations.ExplicitGRIBIUnderNetworkInstance(args.dut) {
		nhg, op2 := gribi.NHGEntry(virtualIPNHGID, map[uint64]uint64{p3NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddEntries(t, []fluent.GRIBIEntry{nh, nhg}, []*client.OpResult{op3, op2})
	} else {
		args.client.AddNHG(t, virtualIPNHGID, map[uint64]uint64{p3NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
	}
	validateTrafficFlows(t, args.ate, []gosnappi.Flow{p3Flow}, []gosnappi.Flow{p2Flow}, nil, pMACFilter)

	t.Logf("Performing implicit in-place replace using the next-hop with ID %v", p2NHID)
	if !deviations.ExplicitGRIBIUnderNetworkInstance(args.dut) {
		nhg, op2 := gribi.NHGEntry(virtualIPNHGID, map[uint64]uint64{p2NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
		args.client.AddEntries(t, []fluent.GRIBIEntry{nh, nhg}, []*client.OpResult{op1, op2})
	} else {
		args.client.AddNHG(t, virtualIPNHGID, map[uint64]uint64{p2NHID: 1}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
	}
	validateTrafficFlows(t, args.ate, []gosnappi.Flow{p2Flow}, []gosnappi.Flow{p3Flow}, nil, pMACFilter)
}

// configureDUT configures DUT.
func configureDUT(t *testing.T, dut *ondatra.DUTDevice) {
	d := gnmi.OC()

	p1 := dut.Port(t, "port1")
	p2 := dut.Port(t, "port2")
	p3 := dut.Port(t, "port3")
	p4 := dut.Port(t, "port4")

	vrf := &oc.NetworkInstance{
		Name: ygot.String(vrfName),
		Type: oc.NetworkInstanceTypes_NETWORK_INSTANCE_TYPE_L3VRF,
	}

	p1VRF := vrf.GetOrCreateInterface(p1.Name())
	p1VRF.Interface = ygot.String(p1.Name())
	p1VRF.Subinterface = ygot.Uint32(0)

	// For interface configuration, Arista prefers config Vrf first then the IP address
	if deviations.InterfaceConfigVRFBeforeAddress(dut) {
		gnmi.Replace(t, dut, gnmi.OC().NetworkInstance(vrfName).Config(), vrf)
	}

	gnmi.Update(t, dut, d.Interface(p1.Name()).Config(), dutPort1.NewOCInterface(p1.Name(), dut))

	if !deviations.InterfaceConfigVRFBeforeAddress(dut) {
		gnmi.Replace(t, dut, gnmi.OC().NetworkInstance(vrfName).Config(), vrf)
	}

	gnmi.Update(t, dut, d.Interface(p2.Name()).Config(), dutPort2.NewOCInterface(p2.Name(), dut))
	gnmi.Update(t, dut, d.Interface(p3.Name()).Config(), dutPort3.NewOCInterface(p3.Name(), dut))
	gnmi.Update(t, dut, d.Interface(p4.Name()).Config(), dutPort4.NewOCInterface(p4.Name(), dut))

	if deviations.ExplicitIPv6EnableForGRIBI(dut) {
		gnmi.Update(t, dut, d.Interface(p2.Name()).Subinterface(0).Ipv6().Enabled().Config(), true)
		gnmi.Update(t, dut, d.Interface(p3.Name()).Subinterface(0).Ipv6().Enabled().Config(), true)
		gnmi.Update(t, dut, d.Interface(p4.Name()).Subinterface(0).Ipv6().Enabled().Config(), true)
	}
	if deviations.ExplicitPortSpeed(dut) {
		fptest.SetPortSpeed(t, p1)
		fptest.SetPortSpeed(t, p2)
		fptest.SetPortSpeed(t, p3)
		fptest.SetPortSpeed(t, p4)
	}
	if deviations.ExplicitInterfaceInDefaultVRF(dut) {
		fptest.AssignToNetworkInstance(t, dut, p2.Name(), deviations.DefaultNetworkInstance(dut), 0)
		fptest.AssignToNetworkInstance(t, dut, p3.Name(), deviations.DefaultNetworkInstance(dut), 0)
		fptest.AssignToNetworkInstance(t, dut, p4.Name(), deviations.DefaultNetworkInstance(dut), 0)
	}
	if deviations.ExplicitGRIBIUnderNetworkInstance(dut) {
		fptest.EnableGRIBIUnderNetworkInstance(t, dut, deviations.DefaultNetworkInstance(dut))
		fptest.EnableGRIBIUnderNetworkInstance(t, dut, vrfName)
	}

	if deviations.GRIBIMACOverrideWithStaticARP(dut) {
		staticARPWithSecondaryIP(t, dut, false)
	}
	if deviations.GRIBIMACOverrideStaticARPStaticRoute(dut) {
		staticARPWithMagicUniversalIP(t, dut)
	}
}

// staticARPWithSecondaryIP configures secondary IPs and static ARP.
func staticARPWithSecondaryIP(t *testing.T, dut *ondatra.DUTDevice, trunk bool, opts ...*bundleName) {
	t.Helper()
	if !trunk {
		p2 := dut.Port(t, "port2")
		p3 := dut.Port(t, "port3")
		gnmi.Update(t, dut, gnmi.OC().Interface(p2.Name()).Config(), dutPort2DummyIP.NewOCInterface(p2.Name(), dut))
		gnmi.Update(t, dut, gnmi.OC().Interface(p3.Name()).Config(), dutPort3DummyIP.NewOCInterface(p3.Name(), dut))
		gnmi.Update(t, dut, gnmi.OC().Interface(p2.Name()).Config(), configStaticArp(p2.Name(), atePort2DummyIP.IPv4, pMAC, false))
		gnmi.Update(t, dut, gnmi.OC().Interface(p3.Name()).Config(), configStaticArp(p3.Name(), atePort3DummyIP.IPv4, pMAC, false))
	} else {
		for _, opt := range opts {

			i2 := &oc.Interface{Name: ygot.String(opt.trunk2)}
			i2.Type = oc.IETFInterfaces_InterfaceType_ieee8023adLag

			i3 := &oc.Interface{Name: ygot.String(opt.trunk3)}
			i3.Type = oc.IETFInterfaces_InterfaceType_ieee8023adLag

			i4 := &oc.Interface{Name: ygot.String(opt.trunk4)}
			i4.Type = oc.IETFInterfaces_InterfaceType_ieee8023adLag

			gnmi.Update(t, dut, gnmi.OC().Interface(opt.trunk2).Config(), configStaticArp(*i2.Name, atePort2.IPv4, port2mac, true))
			gnmi.Update(t, dut, gnmi.OC().Interface(opt.trunk3).Config(), configStaticArp(*i3.Name, atePort3.IPv4, port3mac, true))
			gnmi.Update(t, dut, gnmi.OC().Interface(opt.trunk4).Config(), configStaticArp(*i4.Name, atePort4.IPv4, port4mac, true))
		}
	}
}

func staticARPWithMagicUniversalIP(t *testing.T, dut *ondatra.DUTDevice) {
	t.Helper()
	p2 := dut.Port(t, "port2")
	p3 := dut.Port(t, "port3")
	s2 := &oc.NetworkInstance_Protocol_Static{
		Prefix: ygot.String(atePort2DummyIP.IPv4CIDR()),
		NextHop: map[string]*oc.NetworkInstance_Protocol_Static_NextHop{
			strconv.Itoa(p2NHID): {
				Index: ygot.String(strconv.Itoa(p2NHID)),
				InterfaceRef: &oc.NetworkInstance_Protocol_Static_NextHop_InterfaceRef{
					Interface: ygot.String(p2.Name()),
				},
			},
		},
	}
	s3 := &oc.NetworkInstance_Protocol_Static{
		Prefix: ygot.String(atePort3DummyIP.IPv4CIDR()),
		NextHop: map[string]*oc.NetworkInstance_Protocol_Static_NextHop{
			strconv.Itoa(p3NHID): {
				Index: ygot.String(strconv.Itoa(p3NHID)),
				InterfaceRef: &oc.NetworkInstance_Protocol_Static_NextHop_InterfaceRef{
					Interface: ygot.String(p3.Name()),
				},
			},
		},
	}
	sp := gnmi.OC().NetworkInstance(deviations.DefaultNetworkInstance(dut)).Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_STATIC, deviations.StaticProtocolName(dut))
	gnmi.Replace(t, dut, sp.Static(atePort2DummyIP.IPv4CIDR()).Config(), s2)
	gnmi.Replace(t, dut, sp.Static(atePort3DummyIP.IPv4CIDR()).Config(), s3)
	gnmi.Update(t, dut, gnmi.OC().Interface(p2.Name()).Config(), configStaticArp(p2.Name(), atePort2DummyIP.IPv4, pMAC, false))
	gnmi.Update(t, dut, gnmi.OC().Interface(p3.Name()).Config(), configStaticArp(p3.Name(), atePort3DummyIP.IPv4, pMAC, false))
}

// configStaticArp configures static arp entries
func configStaticArp(p string, ipv4addr string, macAddr string, trunk bool) *oc.Interface {
	i := &oc.Interface{Name: ygot.String(p)}
	if trunk {
		i.Type = oc.IETFInterfaces_InterfaceType_ieee8023adLag
	} else {
		i.Type = oc.IETFInterfaces_InterfaceType_ethernetCsmacd
	}
	s := i.GetOrCreateSubinterface(0)
	s4 := s.GetOrCreateIpv4()
	n4 := s4.GetOrCreateNeighbor(ipv4addr)
	n4.LinkLayerAddress = ygot.String(macAddr)
	return i
}

// TE3.7 case2 - Drain Implementation test.
func testImplementDrain(ctx context.Context, t *testing.T, args *testArgs) {
	if deviations.GRIBIMACOverrideWithStaticARP(args.dut) {
		t.Skip()
		//Testcase skipped as static arp and route config needed for other vendors
	}
	configDUTDrain(t, args.dut)
	addStaticRoute(t, args.dut)

	t.Logf("Adding NHG %d, NH %d and NH %d  via gRIBI", nhg1ID, nh1ID, nh2ID)

	if deviations.GRIBIMACOverrideWithStaticARP(args.dut) {
		args.client.AddNH(t, nh1ID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: btrunk2, Mac: port2mac, Dest: atePort2.IPv4})
		args.client.AddNH(t, nh2ID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: btrunk3, Mac: port3mac, Dest: atePort3.IPv4})

	} else {
		args.client.AddNH(t, nh1ID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: btrunk2, Mac: port2mac})
		args.client.AddNH(t, nh2ID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: btrunk3, Mac: port3mac})

	}
	args.client.AddNHG(t, nhg1ID, map[uint64]uint64{nh1ID: 50, nh2ID: 50}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)

	t.Logf("Adding %d ipv4 vrf prefixes and gribi entries for them", ipv4Prefixcount)

	prefixes := []string{}
	for s := 0; s < ipv4Prefixcount; s++ {
		prefixes = append(prefixes, generateIPAddress(dstPfxvalue, s))
	}

	for _, prefix := range prefixes {
		ipv4Entry := fluent.IPv4Entry().
			WithNetworkInstance(deviations.DefaultNetworkInstance(args.dut)).
			WithPrefix(prefix + "/" + mask).
			WithNextHopGroup(uint64(nhg1ID))
		args.client.Fluent(t).Modify().AddEntry(t, ipv4Entry)
	}
	if err := args.client.AwaitTimeout(context.Background(), t, 2*time.Minute); err != nil {
		t.Fatalf("Error waiting to add IPv4: %v", err)
	}

	gr, err := args.client.Fluent(t).Get().
		WithNetworkInstance(deviations.DefaultNetworkInstance(args.dut)).
		WithAFT(fluent.IPv4).
		Send()

	if err != nil {
		t.Fatalf("got unexpected error from get, got: %v", err)
	}

	for _, prefix := range prefixes {
		chk.GetResponseHasEntries(t, gr,
			fluent.IPv4Entry().
				WithNetworkInstance(deviations.DefaultNetworkInstance(args.dut)).
				WithNextHopGroup(uint64(nhg1ID)).
				WithPrefix(prefix+"/"+mask),
		)
	}

	t.Logf("Adding NHG %d with NH %d as decap and DEFAULT vrf lookup via gRIBI", nhg10ID, nh10ID)
	args.client.AddNH(t, nh10ID, "Decap", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{VrfName: deviations.DefaultNetworkInstance(args.dut)})
	args.client.AddNHG(t, nhg10ID, map[uint64]uint64{nh10ID: 100}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)

	t.Logf("Adding NHG %d with NH %d via gRIBI", nhg20ID, nh20ID)

	args.client.AddNH(t, nh20ID, vip1, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB)
	args.client.AddNHG(t, nhg20ID, map[uint64]uint64{nh20ID: 100}, deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHGOptions{BackupNHG: nhg10ID})
	t.Logf("Adding %d ipv4 vrf prefixes and gribi entries for them", vrfPrefixcount)

	prefixes = []string{}
	for s := 0; s < vrfPrefixcount; s++ {
		prefixes = append(prefixes, generateIPAddress(dstPfxvrf1, s))
	}
	for _, prefix := range prefixes {
		ipv4Entry := fluent.IPv4Entry().
			WithNetworkInstance(vrfName).
			WithPrefix(prefix + "/" + mask).
			WithNextHopGroup(uint64(nhg20ID)).
			WithNextHopGroupNetworkInstance((deviations.DefaultNetworkInstance(args.dut)))
		args.client.Fluent(t).Modify().AddEntry(t, ipv4Entry)
	}
	if err := args.client.AwaitTimeout(context.Background(), t, 2*time.Minute); err != nil {
		t.Fatalf("Error waiting to add IPv4: %v", err)
	}

	grv, err := args.client.Fluent(t).Get().
		WithNetworkInstance(vrfName).
		WithAFT(fluent.IPv4).
		Send()

	if err != nil {
		t.Fatalf("got unexpected error from get, got: %v", err)
	}

	for _, prefix := range prefixes {
		chk.GetResponseHasEntries(t, grv,
			fluent.IPv4Entry().
				WithNetworkInstance(vrfName).
				WithNextHopGroup(uint64(nhg1ID)).
				WithPrefix(prefix+"/"+mask),
		)
	}

	t.Log("Create flows for port1 to port2, port1 to port3 and port1 to port4")

	p2flow := "Port 1 to Port 2"
	p3flow := "Port 1 to Port 3"
	p4flow := "Port 1 to Port 4"
	p2Flow := createFlow(t, p2flow, args.top, &atePort2)
	p3Flow := createFlow(t, p3flow, args.top, &atePort3)
	p4Flow := createFlow(t, p4flow, args.top, &atePort4)

	t.Log("Validate primary path traffic received at ate port2, ate port3 and no traffic on ate port4")
	waitOTGARPEntry(t)
	validateTrafficFlows(t, args.ate, nil, nil, []gosnappi.Flow{p2Flow, p3Flow}, pMACFilter)

	t.Logf("Adding NH %d for trunk4 via gribi", nh3ID)
	if deviations.GRIBIMACOverrideWithStaticARP(args.dut) {
		args.client.AddNH(t, nh3ID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: btrunk4, Mac: port4mac, Dest: atePort4.IPv4})
	} else {
		args.client.AddNH(t, nh3ID, "MACwithInterface", deviations.DefaultNetworkInstance(args.dut), fluent.InstalledInFIB, &gribi.NHOptions{Interface: btrunk4, Mac: port4mac})
	}

	t.Log("Validate traffic switching from  ate port2, ate port3 to ate port4")
	validateTrafficFlows(t, args.ate, []gosnappi.Flow{p4Flow}, []gosnappi.Flow{p2Flow, p3Flow}, nil, pMACFilter)

	t.Log("Validate traffic switching from  ate port4 back to ate port2 and ate port3")
	validateTrafficFlows(t, args.ate, nil, nil, []gosnappi.Flow{p2Flow, p3Flow}, pMACFilter)

}

// deleteinterfaceconfig unconfigs interfaces.
func deleteinterfaceconfig(t *testing.T, dut *ondatra.DUTDevice) {
	d := gnmi.OC()
	p2 := dut.Port(t, "port2")
	p3 := dut.Port(t, "port3")
	p4 := dut.Port(t, "port4")

	if deviations.ExplicitInterfaceInDefaultVRF(dut) {
		ni := deviations.DefaultNetworkInstance(dut)
		gnmi.Delete(t, dut, d.NetworkInstance(ni).Interface(p2.Name()+"."+"0").Subinterface().Config())
		gnmi.Delete(t, dut, d.NetworkInstance(ni).Interface(p3.Name()+"."+"0").Subinterface().Config())
		gnmi.Delete(t, dut, d.NetworkInstance(ni).Interface(p4.Name()+"."+"0").Subinterface().Config())
	} else {
		gnmi.Delete(t, dut, d.Interface(p2.Name()).Subinterface(0).Config())
		gnmi.Delete(t, dut, d.Interface(p3.Name()).Subinterface(0).Config())
		gnmi.Delete(t, dut, d.Interface(p4.Name()).Subinterface(0).Config())
	}
}

// configDUTDrain configures ports for drain test.
func configDUTDrain(t *testing.T, dut *ondatra.DUTDevice) {
	d := gnmi.OC()
	p2 := dut.Port(t, "port2")
	p3 := dut.Port(t, "port3")
	p4 := dut.Port(t, "port4")
	btrunk2 = netutil.NextAggregateInterface(t, dut)

	i2 := &oc.Interface{Name: ygot.String(btrunk2)}
	gnmi.Replace(t, dut, d.Interface(*i2.Name).Config(), configInterfaceDUT(i2, &dutPort2, dut))
	T2 := configureBundle(t, p2.Name(), *i2.Name)
	gnmi.Replace(t, dut, gnmi.OC().Interface(p2.Name()).Config(), T2)

	btrunk3 = netutil.NextAggregateInterface(t, dut)
	i3 := &oc.Interface{Name: ygot.String(btrunk3)}
	gnmi.Replace(t, dut, d.Interface(*i3.Name).Config(), configInterfaceDUT(i3, &dutPort3, dut))
	T3 := configureBundle(t, p3.Name(), *i3.Name)
	gnmi.Replace(t, dut, gnmi.OC().Interface(p3.Name()).Config(), T3)

	btrunk4 = netutil.NextAggregateInterface(t, dut)
	i4 := &oc.Interface{Name: ygot.String(btrunk4)}
	gnmi.Replace(t, dut, d.Interface(*i4.Name).Config(), configInterfaceDUT(i4, &dutPort4, dut))
	T4 := configureBundle(t, p4.Name(), *i4.Name)
	gnmi.Replace(t, dut, gnmi.OC().Interface(p4.Name()).Config(), T4)
	if deviations.ExplicitIPv6EnableForGRIBI(dut) {
		gnmi.Update(t, dut, d.Interface(*i2.Name).Subinterface(0).Ipv6().Enabled().Config(), true)
		gnmi.Update(t, dut, d.Interface(*i3.Name).Subinterface(0).Ipv6().Enabled().Config(), true)
		gnmi.Update(t, dut, d.Interface(*i4.Name).Subinterface(0).Ipv6().Enabled().Config(), true)
	}
	if deviations.ExplicitInterfaceInDefaultVRF(dut) {
		fptest.AssignToNetworkInstance(t, dut, *i2.Name, deviations.DefaultNetworkInstance(dut), 0)
		fptest.AssignToNetworkInstance(t, dut, *i3.Name, deviations.DefaultNetworkInstance(dut), 0)
		fptest.AssignToNetworkInstance(t, dut, *i4.Name, deviations.DefaultNetworkInstance(dut), 0)
	}
	staticARPWithSecondaryIP(t, dut, true, &bundleName{trunk2: btrunk2, trunk3: btrunk3, trunk4: btrunk4})
}

// configInterfaceDUT configures bundle members.
func configInterfaceDUT(i *oc.Interface, a *attrs.Attributes, dut *ondatra.DUTDevice) *oc.Interface {
	i.Type = oc.IETFInterfaces_InterfaceType_ieee8023adLag
	if deviations.InterfaceEnabled(dut) {
		i.Enabled = ygot.Bool(true)
	}
	s := i.GetOrCreateSubinterface(0)
	s4 := s.GetOrCreateIpv4()
	s4a := s4.GetOrCreateAddress(a.IPv4)
	s4a.PrefixLength = ygot.Uint8(ipv4PrefixLen)
	if deviations.InterfaceEnabled(dut) && !deviations.IPv4MissingEnabled(dut) {
		s4.Enabled = ygot.Bool(true)
	}
	return i
}

// configureBundle configures bundle interfaces.
func configureBundle(t *testing.T, name, bundleID string) *oc.Interface {
	i := &oc.Interface{Name: ygot.String(name)}
	i.Type = oc.IETFInterfaces_InterfaceType_ethernetCsmacd
	e := i.GetOrCreateEthernet()
	e.AggregateId = ygot.String(bundleID)
	return i
}

// generateIPAddress generates ipv4 addresses.
func generateIPAddress(dstP string, i int) string {
	ip := net.ParseIP(dstP)
	ip = ip.To4()
	ip[3] = ip[3] + byte(i%256)
	ip[2] = ip[2] + byte(i/256)
	ip[1] = ip[1] + byte(i/(256*256))
	return ip.String()
}

// addStaticRoute configures static routes.
func addStaticRoute(t *testing.T, dut *ondatra.DUTDevice) {
	d := gnmi.OC()
	s := &oc.Root{}
	static := s.GetOrCreateNetworkInstance(deviations.DefaultNetworkInstance(dut)).GetOrCreateProtocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_STATIC, deviations.StaticProtocolName(dut))
	ipv4Nh := static.GetOrCreateStatic(innerDst).GetOrCreateNextHop("0")
	ipv4Nh1 := static.GetOrCreateStatic(innerDst).GetOrCreateNextHop("1")
	ipv4Nh2 := static.GetOrCreateStatic(innerDst).GetOrCreateNextHop("2")
	ipv4Nh.NextHop, _ = ipv4Nh.To_NetworkInstance_Protocol_Static_NextHop_NextHop_Union(atePort2.IPv4)
	ipv4Nh1.NextHop, _ = ipv4Nh.To_NetworkInstance_Protocol_Static_NextHop_NextHop_Union(atePort3.IPv4)
	ipv4Nh2.NextHop, _ = ipv4Nh.To_NetworkInstance_Protocol_Static_NextHop_NextHop_Union(atePort4.IPv4)
	gnmi.Update(t, dut, d.NetworkInstance(deviations.DefaultNetworkInstance(dut)).Protocol(oc.PolicyTypes_INSTALL_PROTOCOL_TYPE_STATIC, deviations.StaticProtocolName(dut)).Config(), static)
}

func configureATE(t *testing.T, ate *ondatra.ATEDevice) gosnappi.Config {
	top := ate.OTG().NewConfig(t)

	p1 := ate.Port(t, "port1")
	p2 := ate.Port(t, "port2")
	p3 := ate.Port(t, "port3")
	p4 := ate.Port(t, "port4")

	atePort1.AddToOTG(top, p1, &dutPort1)
	atePort2.AddToOTG(top, p2, &dutPort2)
	atePort3.AddToOTG(top, p3, &dutPort3)
	atePort4.AddToOTG(top, p4, &dutPort4)

	return top
}

// createFlow returns a flow from atePort1 to the dstPfx, expected to arrive on ATE interface dsts.
func createFlow(_ *testing.T, name string, ateTop gosnappi.Config, dsts ...*attrs.Attributes) gosnappi.Flow {
	var rxEndpoints []string
	for _, dst := range dsts {
		rxEndpoints = append(rxEndpoints, dst.Name+".IPv4")
	}

	flowipv4 := ateTop.Flows().Add().SetName(name)
	flowipv4.Metrics().SetEnable(true)
	e1 := flowipv4.Packet().Add().Ethernet()
	e1.Src().SetValue(atePort1.MAC)
	e1.Dst().SetChoice("value").SetValue(pMAC)
	flowipv4.TxRx().Device().SetTxNames([]string{atePort1.Name + ".IPv4"}).SetRxNames(rxEndpoints)
	outerIPHeader := flowipv4.Packet().Add().Ipv4()
	outerIPHeader.Src().SetValue(atePort1.IPv4)
	outerIPHeader.Dst().SetValue(dstPfxFlowIP)
	innerIPHeader := flowipv4.Packet().Add().Ipv4()
	innerIPHeader.Src().Increment().SetStart(innerSrcIPv4Start).SetStep("0.0.0.1").SetCount(ipv4FlowCount)
	innerIPHeader.Dst().Increment().SetStart(innerDstIPv4Start).SetStep("0.0.0.1").SetCount(ipv4FlowCount)
	flowipv4.Size().SetFixed(100)
	eth := flowipv4.EgressPacket().Add().Ethernet()
	ethTag := eth.Dst().MetricTags().Add()
	ethTag.SetName("EgressTrackingFlow").SetOffset(36).SetLength(12)

	return flowipv4
}

// validateTrafficFlows starts traffic and ensures that good flows have 0% loss and bad flows have
// 100% loss.
func validateTrafficFlows(t *testing.T, ate *ondatra.ATEDevice, good, bad, lb []gosnappi.Flow, macFilter string) {

	if len(good) == 0 && len(bad) == 0 && len(lb) == 0 {
		return
	}

	otg := ate.OTG()
	config := otg.FetchConfig(t)
	otg.StartTraffic(t)
	time.Sleep(15 * time.Second)
	otg.StopTraffic(t)

	otgutils.LogFlowMetrics(t, otg, config)
	otgutils.LogPortMetrics(t, otg, config)
	for _, flow := range good {
		rxPkts := gnmi.Get(t, ate.OTG(), gnmi.OTG().Flow(flow.Name()).Counters().InPkts().State())
		if got := getLossPct(t, flow.Name()); got > 0 {
			t.Errorf("LossPct for flow %s: got %v, want 0", flow.Name(), got)
		}
		etPath := gnmi.OTG().Flow(flow.Name()).TaggedMetricAny()
		ets := gnmi.GetAll(t, ate.OTG(), etPath.State())
		if got := len(ets); got != 1 {
			t.Errorf("EgressTracking got %d items, want %d", got, 1)
		}
		etTagspath := gnmi.OTG().Flow(flow.Name()).TaggedMetricAny().TagsAny()
		etTags := gnmi.GetAll(t, ate.OTG(), etTagspath.State())
		if got := etTags[0].GetTagValue().GetValueAsHex(); !strings.EqualFold(got, macFilter) {
			t.Errorf("EgressTracking filter got %q, want %q", got, macFilter)
		}
		if got := ets[0].GetCounters().GetInPkts(); got != rxPkts {
			t.Errorf("EgressTracking counter in-pkts got %d, want %d", got, rxPkts)
		} else {
			t.Logf("Received %d packets with %s as the last 12 bits in the dst MAC", got, macFilter)
		}

	}
	for _, flow := range lb {
		// for LB flows, we expect to receive between 48-52% of packets on each interface (before and after filtering).
		lbPct := 50.0
		if diff := cmp.Diff(float32(lbPct), getLossPct(t, flow.Name()), cmpopts.EquateApprox(0, lbPrecision)); diff != "" {
			t.Errorf("Received number of packets -want,+got:\n%s", diff)
		}
		etPath := gnmi.OTG().Flow(flow.Name()).TaggedMetricAny()
		ets := gnmi.GetAll(t, ate.OTG(), etPath.State())
		if got := len(ets); got != 1 {
			t.Errorf("EgressTracking got %d items, want %d", got, 1)
			return
		}
		etTagspath := gnmi.OTG().Flow(flow.Name()).TaggedMetricAny().TagsAny()
		etTags := gnmi.GetAll(t, ate.OTG(), etTagspath.State())
		if got := etTags[0].GetTagValue().GetValueAsHex(); !strings.EqualFold(got, macFilter) {
			t.Errorf("EgressTracking filter got %q, want %q", got, macFilter)
		}
		inPkts := gnmi.Get(t, ate.OTG(), gnmi.OTG().Flow(flow.Name()).Counters().InPkts().State())
		if diff := cmp.Diff(inPkts, ets[0].GetCounters().GetInPkts(), cmpopts.EquateApprox(lbPct, lbPrecision)); diff != "" {
			t.Errorf("EgressTracking received number of packets -want,+got:\n%s", diff)
		}
	}
	for _, flow := range bad {
		if got := getLossPct(t, flow.Name()); got < 100 {
			t.Errorf("LossPct for flow %s: got %v, want 100", flow.Name(), got)
		}
	}
}

// getLossPct returns the loss percentage for a given flow
func getLossPct(t *testing.T, flowName string) float32 {
	t.Helper()
	otg := ondatra.ATE(t, "ate").OTG()
	flowStats := gnmi.Get(t, otg, gnmi.OTG().Flow(flowName).State())
	txPackets := float32(flowStats.GetCounters().GetOutPkts())
	rxPackets := float32(flowStats.GetCounters().GetInPkts())
	lostPackets := txPackets - rxPackets
	if txPackets == 0 {
		t.Fatalf("Tx packets should be higher than 0 for flow %s", flowName)
	}
	lossPct := lostPackets * 100 / txPackets
	return lossPct
}

// Waits for an ARP entry to be present for ATE Port1
func waitOTGARPEntry(t *testing.T) {
	t.Helper()
	ate := ondatra.ATE(t, "ate")
	gnmi.WatchAll(t, ate.OTG(), gnmi.OTG().Interface(atePort1.Name+".Eth").Ipv4NeighborAny().LinkLayerAddress().State(), time.Minute, func(val *ygnmi.Value[string]) bool {
		return val.IsPresent()
	}).Await(t)
}
