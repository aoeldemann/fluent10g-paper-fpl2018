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
)

func main() {
	// set log level to INFO to reduce verbosity of output
	gofluent10g.LogSetLevel(gofluent10g.LOG_INFO)

	// open network tester
	nt := gofluent10g.NetworkTesterCreate()
	defer nt.Close()

	// get generators
	gens := nt.GetGenerators()

	// iterate over all packet sizes
	for i, pktlen := range pktlens {

		gofluent10g.Log(gofluent10g.LOG_INFO,
			"%d/%d Replay: Datarate: 4x %.2f bps (duplex), "+
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

		// start replay (blocks until replay finished)
		nt.StartReplay()

		gofluent10g.Log(gofluent10g.LOG_INFO, "done! (hardware did not flag "+
			"error, so this was a success!)")

		// free host memory we do not need anymore
		trace = nil
		nt.FreeHostMemory()

		gofluent10g.LogDecrementIndentLevel()
	}

	// no errors occurs, so this was a success!
	gofluent10g.Log(gofluent10g.LOG_INFO, "Successfully maintained replay "+
		"data rate of 4x %.2f bps for the following packet sizes: %d", datarate,
		pktlens)
}
