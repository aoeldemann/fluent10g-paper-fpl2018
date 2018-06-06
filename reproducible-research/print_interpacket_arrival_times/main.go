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
	"github.com/aoeldemann/gofluent10g"
	"github.com/aoeldemann/gofluent10g/utils"
	"sort"
	"time"
)

var (
	// measurement data rates
	datarates = []float64{100e6, 1e9, 5e9, 10e9}

	// meausrement packet sizes
	pktlens = []int{64, 1518}

	// generator interface id
	ifGen = 0

	// receiver interface id
	ifRecv = 1

	// measurement duration
	duration = 10 * time.Second
)

func main() {
	// set log level to INFO to reduce verbosity of output
	gofluent10g.LogSetLevel(gofluent10g.LOG_INFO)

	// open network tester
	nt := gofluent10g.NetworkTesterCreate()
	defer nt.Close()

	// get generator and receivers
	gen := nt.GetGenerator(ifGen)
	recv := nt.GetReceiver(ifRecv)

	// enable packet capture on receiver interface. since we are only
	// interested in the inter-packet arrival times, we disable the capturing
	// of packet data
	recv.EnableCapture(true)
	recv.SetCaptureMaxLen(0)

	// iterate over all data rates
	for i, datarate := range datarates {

		// iterate over all packet sizes
		for j, pktlen := range pktlens {
			gofluent10g.Log(gofluent10g.LOG_INFO, "%d/%d: Datarate: %.2f bps, "+
				"Packet length: %d", (i*len(pktlens) + j + 1),
				len(datarates)*len(pktlens), datarate, pktlen)

			gofluent10g.LogIncrementIndentLevel()

			gofluent10g.Log(gofluent10g.LOG_INFO, "Generating trace ...")

			// generate CBR trace data with fixed packet length. Trace duration
			// is 10 seconds. we only transfer the first 34 bytes of each
			// packet down to hardware (contains ethernet and ipv4 headers),
			// hardware will append zero bytes before transmission to restore
			// the original packet lengths
			trace := utils.GenTraceCBR(datarate, pktlen, 34, duration, 1)

			// assign trace to generator
			gen.SetTrace(trace)

			// calculate the host memory size we need to store the capture data.
			// we only store meta data (8 byte) for each packet, no packet
			// data
			captureMemSize := uint64(trace.GetPacketCount()) * 8

			// set receiver capture host memory size
			recv.SetCaptureHostMemSize(captureMemSize)

			// write config to hardware
			nt.WriteConfig()

			gofluent10g.Log(gofluent10g.LOG_INFO, "Starting replay and capture ...")

			// start capturing
			nt.StartCapture()

			// start replay (blocks until replay finished)
			nt.StartReplay()

			gofluent10g.Log(gofluent10g.LOG_INFO, "Replay done")

			// wait a little to make sure all packets have been captured
			time.Sleep(time.Second)

			// stop capturing
			nt.StopCapture()

			gofluent10g.Log(gofluent10g.LOG_INFO, "Capture done")

			// get capture data structure
			capture := recv.GetCapture()

			// get captured packets
			pkts := capture.GetPackets()

			// make sure all generated packets arrived back at the receiver
			if len(pkts) != trace.GetPacketCount() {
				gofluent10g.Log(gofluent10g.LOG_ERR,
					"not all generated packets arrived back at the receiver")
			}

			gofluent10g.Log(gofluent10g.LOG_INFO,
				"Calculating arrival time statistics ...")

			// get packet arrival times
			arrivalTimes := pkts.GetArrivalTimes()

			// inter-packet arrival time is a relative metric -> value
			// for first packet is not meaningful
			arrivalTimes = arrivalTimes[1 : len(arrivalTimes)-1]

			// get minimum and maximum inter-packet arrival times
			sort.Sort(sort.Float64Slice(arrivalTimes))
			arrivalTimeMin := arrivalTimes[0]
			arrivalTimeMax := arrivalTimes[len(arrivalTimes)-1]

			// convert inter-packet arrival times to nanoseconds
			arrivalTimeMin *= 1e9
			arrivalTimeMax *= 1e9

			// output information
			gofluent10g.Log(gofluent10g.LOG_INFO,
				"Minimum inter-packet arrival time: %.2f ns", arrivalTimeMin)
			gofluent10g.Log(gofluent10g.LOG_INFO,
				"Maxmimum inter-packet arrival time: %.2f ns", arrivalTimeMax)

			// reset pointers pointing to data we do not need anymore
			trace = nil
			capture = nil
			pkts = nil

			// free memory
			nt.FreeHostMemory()

			gofluent10g.LogDecrementIndentLevel()
		}
	}
}
