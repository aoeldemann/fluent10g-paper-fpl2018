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
	"time"
)

var (
	// per-generator data rate
	datarate = 10e9

	// packet sizes
	pktlens = []int{64, 100, 300, 500, 700, 900, 1100, 1300, 1518}

	// measurement duration
	duration = 10 * time.Second

	// generator and receiver interface ids
	ifsGen  = []int{0, 1, 2}
	ifsRecv = []int{0, 1, 3}
)

func main() {
	// set log level to INFO to reduce verbosity of output
	gofluent10g.LogSetLevel(gofluent10g.LOG_INFO)

	// open network tester
	nt := gofluent10g.NetworkTesterCreate()
	defer nt.Close()

	// get generators and receivers
	gens := make(gofluent10g.Generators, 3)
	recvs := make(gofluent10g.Receivers, 3)
	for i := 0; i < 3; i++ {
		gens[i] = nt.GetGenerator(ifsGen[i])
		recvs[i] = nt.GetReceiver(ifsRecv[i])
	}

	// enable packet capture on all receivers. we want to capture entire
	// packets, so set max. capture length to 1518. Discard capture data once
	// it has been transferred from the FPGA to reduce memory footprint
	for _, recv := range recvs {
		recv.EnableCapture(true)
		recv.SetCaptureMaxLen(1518)
		recv.SetCaptureDiscard(true)
	}

	// iterate over all packet sizes
	for i, pktlen := range pktlens {

		gofluent10g.Log(gofluent10g.LOG_INFO,
			"%d/%d Replay + Capture: Datarate: 3x %.2f bps (each), "+
				"Packet Length: %d", (i + 1), len(pktlens), datarate, pktlen)

		gofluent10g.LogIncrementIndentLevel()

		gofluent10g.Log(gofluent10g.LOG_INFO, "Generating trace ...")

		// generate CBR traffic trace
		trace := utils.GenTraceCBR(datarate, pktlen, pktlen-4, duration, 1)

		// assign traces to generators
		for _, gen := range gens {
			gen.SetTrace(trace)
		}

		gofluent10g.Log(gofluent10g.LOG_INFO, "Starting measurement ...")

		// write config to hardware
		nt.WriteConfig()

		// start capturing
		nt.StartCapture()

		// start replay (blocks until replay finished)
		nt.StartReplay()

		// stop capturing
		nt.StopCapture()

		gofluent10g.Log(gofluent10g.LOG_INFO, "done! (hardware did not flag "+
			"error, so this was a success!)")

		// free host memory we do not need anymore
		trace = nil
		nt.FreeHostMemory()

		gofluent10g.LogDecrementIndentLevel()
	}

	// no errors occurs, so this was a success!
	gofluent10g.Log(gofluent10g.LOG_INFO, "Successfully maintained replay and "+
		"capture data rate of 3x %.2f bps for the following packet sizes: %d",
		datarate, pktlens)
}
