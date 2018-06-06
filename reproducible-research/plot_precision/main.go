// The MIT License
//
// Copyright (c) 2017-2018 by the author(s)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//
// Author(s):
//   - Andreas Oeldemann <andreas.oeldemann@tum.de>
//
// Description:
//
// see README.md

package main

import (
	"encoding/binary"
	"fmt"
	"github.com/aoeldemann/gofluent10g"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

var (
	// target mean data rate
	datarateMean = 8e9

	// measurement duration
	duration = 60 * time.Second

	// time between to PTP packet burst insertions
	ptpInterval = 75 * time.Microsecond
)

func genTrace() (*gofluent10g.Trace, []float64) {
	// packet length is uniformly distributed between 64 and 1518 bytes. Since
	// MAC will append FCS, the packets we generate here are 4 bytes shorter
	pktlenMin := 60
	pktlenMax := 1514
	pktlenMean := (pktlenMin + pktlenMax) / 2

	// calculate the average time of the gap between two packets (add 24 bytes
	// for FCS, preamble, SOD and inter-frame gap
	tGapMean := float64(8*(pktlenMean+24))/datarateMean -
		float64(8*(pktlenMean+24))/10e9

	// calculate the number of packets we will generate. add 24 bytes to the
	// packet length to account for Ethernet preamble + SOD, inter-frame gap
	// and FCS
	nPkts := round(duration.Seconds() * datarateMean /
		float64(8*(pktlenMean+24)))

	gofluent10g.Log(gofluent10g.LOG_INFO, "Generating %d packets", nPkts)

	// we will reuse the same ethernet header for all packets, generate source
	// and destination MAC addresses
	macSrc, _ := net.ParseMAC("53:00:00:00:00:01")
	macDst, _ := net.ParseMAC("53:00:00:00:00:02")

	// generate ethernet header
	hdrEth := &layers.Ethernet{
		SrcMAC: macSrc,
		DstMAC: macDst,
	}

	// serialize packet data
	bufPkt := gopacket.NewSerializeBuffer()
	err := gopacket.SerializeLayers(bufPkt, gopacket.SerializeOptions{},
		hdrEth)
	if err != nil {
		gofluent10g.Log(gofluent10g.LOG_ERR, "%s", err.Error())
	}
	bufPktData := bufPkt.Bytes()[0:16]

	// calculate total amount of trace data we will need to write to the
	// hardware. for each packet we transfer 8 bytes of meta data and 16 bytes
	// of packet data (14 bytes for the ethernet header, two more for the PTP
	// header fields we need to set)
	bufTraceSize := 24 * uint64(nPkts)

	// trace data must be aligned to 64 byte
	if bufTraceSize%64 != 0 {
		bufTraceSize = 64 * (bufTraceSize/64 + 1)
	}

	// allocate memory
	bufTrace := make([]byte, bufTraceSize)

	// accumulated inter-packet clock cycle rounding error
	accCyclesInterPacketRoundErr := 0.0

	// accumulated number of clock cycles between packets
	accCyclesInterPacket := uint64(0)

	// create data structure to save inter-packet times of ptp packets
	var tInterPacketsPTP []float64

	// convert ptp interval time to clock cycles
	ptpIntervalCycles := int(ptpInterval.Nanoseconds() * gofluent10g.FREQ_SFP /
		1e9)

	// true if a burst of four ptp packets is currently being inserted in the
	// trace
	ptpBurstActive := false

	// counts the number of cycles between to ptp packet bursts
	ptpInterPacketCycleCounter := 0

	// counts the number of cycles of ptp packets in a burst (0-4)
	ptpBurstPacketCounter := 0

	// counts the total number of ptp packets that have been inserted in the
	// trace
	ptpPacketCounter := 0

	for i := 0; i < nPkts; i++ {
		// determine packet length according to uniform distribution
		lenWire := rand.Intn(pktlenMax-pktlenMin+1) + pktlenMin

		// calculate the time it takes to transmit this packet
		tTransfer := float64(8*(lenWire+24)) / 10e9

		// get a random gap between this and the next packet
		tGap := tGapMean * rand.ExpFloat64()

		// both times added up -> inter-packet time
		tInterPacket := tTransfer + tGap

		// hardware does not support inter-packet times larger than 2**32-1 *
		// T_CLK, so cut if necessary
		if tInterPacket > 4294967295/gofluent10g.FREQ_SFP {
			tInterPacket = 4294967295 / gofluent10g.FREQ_SFP
		}

		// caculate the number of cycles between packets (do not round yet)
		cyclesInterPacket := tInterPacket * gofluent10g.FREQ_SFP

		// the number of clock cycles between two packets is a floating-point
		// number, but clock cycles must always be integer values. If we
		// always round up we are sending too slow, if we always round down
		// we are sending too fast. Sending too fast at full line-rate causes
		// timing errors (we cannot send faster than 10 Gbps!). We start by
		// rounding up and accumulate the resulting rounding error. If the
		// accumulated error becomes larger than one full clock cycle, we
		// round down and decrease the accumulated error. On average we will
		// hit the target mean data rate.
		if accCyclesInterPacketRoundErr < 1.0 {
			// not enough rounding error accumulated yet -> round up
			accCyclesInterPacketRoundErr +=
				math.Ceil(cyclesInterPacket) - cyclesInterPacket
			cyclesInterPacket = math.Ceil(cyclesInterPacket)
		} else {
			// enough rounding error accumulated -> round down
			accCyclesInterPacketRoundErr -=
				cyclesInterPacket - math.Floor(cyclesInterPacket)
			cyclesInterPacket = math.Floor(cyclesInterPacket)
		}

		// accumulate clock cycles between all packets
		accCyclesInterPacket += uint64(cyclesInterPacket)

		if ptpBurstActive || (ptpInterPacketCycleCounter > ptpIntervalCycles) {
			// ether we must insert a ptp packet because a ptp burst is
			// currently active or because the number of clock cycles between
			// to ptp bursts is reached

			if ptpBurstPacketCounter == 0 {
				// starts a new burst
				ptpBurstActive = true
			}

			// set ethertype to ieee1588 (0x88F7), changed byte-order
			binary.LittleEndian.PutUint16(bufPktData[12:14], 0xf788)

			// set ptp version to 2
			binary.LittleEndian.PutUint16(bufPktData[15:17], 0x2)

			// save inter-packet time
			if ptpBurstPacketCounter != 3 {
				tInterPacketsPTP = append(tInterPacketsPTP, tInterPacket)
			}

			// increment counters
			ptpPacketCounter++
			ptpBurstPacketCounter++

			if ptpBurstPacketCounter == 4 {
				// burst is over
				ptpBurstActive = false
				ptpBurstPacketCounter = 0
			}
			ptpInterPacketCycleCounter = 0
		} else {
			// this is not a ptp packet, so set ethertype to ipv4 (reverse
			// byte order)
			binary.LittleEndian.PutUint16(bufPktData[12:14], 0x0008)
		}

		ptpInterPacketCycleCounter += int(cyclesInterPacket)

		// assemble meta data
		meta := uint64(cyclesInterPacket)
		meta |= uint64(16) << 32 // capture length. eth header + 2 bytes ptp
		meta |= uint64(lenWire) << 48

		// write meta data
		binary.LittleEndian.PutUint64(bufTrace[i*24:i*24+8], meta)

		// write packet data
		copy(bufTrace[i*24+8:i*24+24], bufPktData)
	}

	// add padding for 64 byte alignment
	addr := nPkts * 24
	for addr%64 != 0 {
		binary.LittleEndian.PutUint64(bufTrace[addr:addr+8], 0xFFFFFFFFFFFFFFFF)
		addr += 8
	}

	// calculate actual replay duration after rounding and print it
	actualDuration :=
		time.Duration(float64(accCyclesInterPacket)/gofluent10g.FREQ_SFP*1e9) *
			time.Nanosecond
	gofluent10g.Log(gofluent10g.LOG_INFO,
		"Actual trace duration: %s (Target was %s)",
		actualDuration, duration)

	gofluent10g.Log(gofluent10g.LOG_INFO, "Generated packets: %d", nPkts)
	gofluent10g.Log(gofluent10g.LOG_INFO, "Generated PTP packets: %d",
		ptpPacketCounter)

	// create and return trace
	return gofluent10g.TraceCreateFromData(bufTrace, nPkts, actualDuration, 1), tInterPacketsPTP
}

func main() {
	// create a random traffic trace (uniform distributed packet sizes,
	// expontentially distributed inter-packet gaps) with a mean data rate of
	// datarateMean. Every ptpInterval, four packets are replaced by PTP
	// packets for inter-packet time measurements.
	trace, tInterPacketsPTP := genTrace()

	// set output filename
	filename := "output/timestamp_diffs_expected.dat"

	// open file to write inter-packet times of ptp packets
	file, err := os.Create(filename)
	if err != nil {
		gofluent10g.Log(gofluent10g.LOG_ERR, "could not create file '%s'",
			filename)
		return
	}
	defer file.Close()

	gofluent10g.Log(gofluent10g.LOG_INFO,
		"Writing expected inter-packet times to output file '%s' ...", filename)

	// write inter-packet times to file
	for _, tInterPacket := range tInterPacketsPTP {
		file.WriteString(fmt.Sprintf("%.12f\n", tInterPacket))
	}

	// open network tester
	nt := gofluent10g.NetworkTesterCreate()
	defer nt.Close()

	// assign trace to generator on interface 0
	nt.GetGenerator(0).SetTrace(trace)

	// write network tester configuration
	nt.WriteConfig()

	// start replay
	nt.StartReplay()
}

func round(x float64) int {
	return int(math.Floor(x + 0.5))
}
