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
	"os"
	"time"
)

var (
	// per-generator data rate (bisection start point, initial step size and
	// abort condition)
	datarateMin       = 8e9
	datarateStepInit  = 2e9
	datarateStepLimit = 0.01e9

	// packet sizes
	pktlens = []int{64, 104, 152, 200, 256, 304, 352, 400, 456, 504, 552, 600,
		656, 704, 752, 800, 856, 904, 952, 1000, 1056, 1104, 1152, 1200, 1256,
		1304, 1352, 1400, 1456, 1518}

	// measurement duration
	duration = 10 * time.Second
)

func main() {
	// set log level to INFO to reduce verbosity of output
	gofluent10g.LogSetLevel(gofluent10g.LOG_INFO)

	// open network tester
	nt := gofluent10g.NetworkTesterCreate()
	defer nt.Close()

	// when the hardware is unable to replay/capture data fast enough it
	// sets an error register and stops operation. The library continuously
	// checks this error register and aborts the application when an error is
	// detected. Since we want to determine the maximum replay + capture
	// throughput the network tester is able to sustain, we want to increase
	// the data rate step by step and find the first measurement point where
	// the hardware's error registers are set. We do not want the application
	// to exit, because the error is expected. Thus, we disable automatic
	// error checking and check manually ourselves after each measurement run
	nt.SetCheckErrors(false)

	// get generators and receiver
	gens := nt.GetGenerators()
	recvs := nt.GetReceivers()

	// enable packet capture on all receivers. Discard capture data once it has
	// been transferred from the FPGA to reduce memory footprint
	for _, recv := range recvs {
		recv.EnableCapture(true)
		recv.SetCaptureDiscard(true)
	}

	// open output file for writing
	filename := "output/max_throughput.dat"
	file, err := os.Create(filename)
	if err != nil {
		gofluent10g.Log(gofluent10g.LOG_ERR, "could not create file '%s'",
			filename)
	}
	defer file.Close()

	gofluent10g.Log(gofluent10g.LOG_INFO, "Writing results to file '%s'",
		filename)

	// set max capture length to 1518
	for _, recv := range recvs {
		recv.SetCaptureMaxLen(1518)
	}

	// iterate over all packet sizes
	for i, pktlen := range pktlens {

		// initialize bisection values
		datarate := datarateMin
		datarateStep := datarateStepInit
		memBandwidthMax := 0.0
		datarateMax := 0.0

		for {
			gofluent10g.Log(gofluent10g.LOG_INFO,
				"%d/%d: Datarate: 4x %.2f bps, Packet Length: %d",
				i+1, len(pktlens), datarate, pktlen)

			gofluent10g.LogIncrementIndentLevel()
			gofluent10g.Log(gofluent10g.LOG_INFO, "Generating trace ...")

			// generate CBR traffic trace
			trace := utils.GenTraceCBR(datarate, pktlen, pktlen-4,
				duration, 1)

			// calculate required memory bandwidth to write the trace
			// to memory (per network interface, per memory read/write
			// direction)
			memBandwidth := 8.0 * float64(trace.GetSize()) /
				trace.GetDuration().Seconds()

			// assign traces to generators
			for _, gen := range gens {
				gen.SetTrace(trace)
			}

			gofluent10g.Log(gofluent10g.LOG_INFO, "Performing measurement ...")

			// write config to hardware
			nt.WriteConfig()

			// start capturing
			nt.StartCapture()

			// start replay (blocks until replay finished)
			nt.StartReplay()

			// stop capturing
			nt.StopCapture()

			// has throughput limit been reached?
			var limitReached bool

			if err := nt.CheckErrors(); err != nil {
				// hardware flagged an error, so throughput limit is reached
				limitReached = true
				gofluent10g.Log(gofluent10g.LOG_INFO, "Throughput limit "+
					"reached. Hardware asserted the error: '%s'", err.Error())
			} else {
				// limit has not been reached yet, so make sure all packets
				// that were generated arrived back at the network tester
				limitReached = false

				// get total number of sent and captured packets
				nPktsTotalTX := 0
				nPktsTotalCaptured := 0
				for i := 0; i < 4; i++ {
					nPktsTotalTX += nt.GetInterface(i).GetPacketCountTX()
					nPktsTotalCaptured += recvs[i].GetPacketCountCaptured()
				}

				// make sure number of sent and captured packets matches
				if nPktsTotalTX != nPktsTotalCaptured {
					gofluent10g.Log(gofluent10g.LOG_ERR,
						"nPktsTotalTX != nPktsTotalCaptured")
				}

				if nPktsTotalTX != 4*trace.GetPacketCount() {
					gofluent10g.Log(gofluent10g.LOG_ERR,
						"not all trace packets have been replayed")
				}
			}

			// keep track of the maximum data rate and packets per second
			// we can achieve
			if (limitReached == false) && (datarate > datarateMax) {
				datarateMax = datarate
				memBandwidthMax = memBandwidth
			}

			if datarateStep <= datarateStepLimit || datarateMax >= 10e9 {
				// bisection abort condition satisfied

				gofluent10g.Log(gofluent10g.LOG_INFO,
					"--> Throughput Limit: 4x %.2f bps, "+
						"Memory bandwidth: 4x %.2f",
					datarateMax, memBandwidthMax)

				// we are generating and capturing data on four network
				// interfaces, so multiply datarate by four
				datarateMax *= 4.0

				// calculate total memory bandwidth:
				// --> 4 network interfaces: 4x
				// --> concurrent replay and capture: 2x
				// --> reading + writing from DRAM: 2x
				// ----> 16x
				memBandwidthMax *= 16.0

				// write results to output file
				file.WriteString(fmt.Sprintf("%d %f %f\n", pktlen,
					datarateMax, memBandwidthMax))

				// done for this packet length
				gofluent10g.LogDecrementIndentLevel()
				break
			} else {
				// select next data rate to measure
				if limitReached {
					datarate -= datarateStep
				} else {
					datarate += datarateStep
				}
				datarateStep /= 2.0
			}

			// free host memory we do not need anymore
			trace = nil
			nt.FreeHostMemory()

			gofluent10g.LogDecrementIndentLevel()
		}
	}
}
