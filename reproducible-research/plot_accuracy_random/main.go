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
	"fmt"
	"github.com/aoeldemann/gofluent10g"
	"github.com/aoeldemann/gofluent10g/utils"
	"math/rand"
	"os"
	"sort"
	"time"
)

var (
	// mean measurement data rates
	dataratesMean = []float64{1e9, 5e9, 8e9}

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

	// seed the random number generator
	rand.Seed(time.Now().UTC().UnixNano())

	// open network tester
	nt := gofluent10g.NetworkTesterCreate()
	defer nt.Close()

	// get generator and receivers
	gen := nt.GetGenerator(ifGen)
	recv := nt.GetReceiver(ifRecv)

	// enable packet capture on receiver interface. since we are only
	// interested in packet latency, we disable the capturing of packet data
	recv.EnableCapture(true)
	recv.SetCaptureMaxLen(0)

	// set up timestamping
	nt.SetTimestampMode(gofluent10g.TimestampModeFixedPos)
	nt.SetTimestampPos(0)
	nt.SetTimestampWidth(24)

	// iterate over all mean data rates
	for i, datarateMean := range dataratesMean {

		gofluent10g.Log(gofluent10g.LOG_INFO, "%d/%d: Mean Datarate: %.2f bps",
			i+1, len(dataratesMean), datarateMean)

		gofluent10g.LogIncrementIndentLevel()

		gofluent10g.Log(gofluent10g.LOG_INFO, "Generating trace ...")

		// generate random traffic trace with a given mean data rate. packet
		// sizes are uniform distributed between 64 and 1518 bytes, frame
		// is generated according to exponential distribution. We only transfer
		// the first 34 bytes of each packet down to hardware (contains
		// ethernet and ipv4 headers), hardware will append zero bytes before
		// transmission to restore the original packet length
		trace := utils.GenTraceRandom(datarateMean, 34, duration, 1)

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
			"Calculating latency statistics ...")

		// calculate latency mean and std dev
		latencyMean := utils.CalcLatencyMean(pkts)
		latencyStd := utils.CalcLatencyStdDev(pkts, latencyMean)

		// sort packets in ascending latency order
		sort.Sort(gofluent10g.CapturePacketsSortByLatency(pkts))

		// calculate latency histogram
		latencyHistogram, _ := utils.CalcLatencyHistogram(pkts)

		// get minimum and maximum latency in nanoseconds
		latencyMin := pkts[0].Latency * 1e9
		latencyMax := pkts[len(pkts)-1].Latency * 1e9

		// calculate latency mean and stddev in nanoseconds
		latencyMean *= 1e9
		latencyStd *= 1e9

		// output some infos
		gofluent10g.Log(gofluent10g.LOG_INFO, "Captured %d packets.", len(pkts))
		gofluent10g.Log(gofluent10g.LOG_INFO, "Min latency: %.2f ns",
			latencyMin)
		gofluent10g.Log(gofluent10g.LOG_INFO, "Max latency: %.2f ns",
			latencyMax)
		gofluent10g.Log(gofluent10g.LOG_INFO, "Mean latency: %.2f ns",
			latencyMean)
		gofluent10g.Log(gofluent10g.LOG_INFO, "Stddev latency: %.2f ns",
			latencyStd)

		// assemble output filename for this run
		filename := fmt.Sprintf("output/histogram_%d.dat", int(datarateMean))

		// open output file for writing
		file, err := os.Create(filename)
		if err != nil {
			gofluent10g.Log(gofluent10g.LOG_ERR, "could not create file '%s'",
				filename)
			return
		}
		defer file.Close()

		gofluent10g.Log(gofluent10g.LOG_INFO,
			"Writing latency histogram to output file '%s' ...", filename)

		// write historam values (after conversion to nanoseconds) to file
		for _, elem := range latencyHistogram {
			file.WriteString(fmt.Sprintf("%f %d\n", elem.Latency*1e9,
				elem.Occurrences))
		}

		// reset pointers pointing to data we do not need anymore
		trace = nil
		capture = nil
		pkts = nil

		// free memory
		nt.FreeHostMemory()

		gofluent10g.LogDecrementIndentLevel()
	}
}
