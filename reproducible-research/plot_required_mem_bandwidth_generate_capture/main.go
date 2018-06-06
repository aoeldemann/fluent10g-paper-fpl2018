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
	"runtime"
	"time"
)

var (
	// data rate
	datarate = 10e9

	// packet sizes
	pktlens = []int{64, 104, 152, 200, 256, 304, 352, 400, 456, 504, 552, 600,
		656, 704, 752, 800, 856, 904, 952, 1000, 1056, 1104, 1152, 1200, 1256,
		1304, 1352, 1400, 1456, 1518}

	// trace duration
	duration = 10 * time.Second
)

func main() {
	// set log level to INFO to reduce verbosity of output
	gofluent10g.LogSetLevel(gofluent10g.LOG_INFO)

	// open output file for writing
	filename := "output/required_membandwidth.dat"
	file, err := os.Create(filename)
	if err != nil {
		gofluent10g.Log(gofluent10g.LOG_ERR, "could not create file '%s'",
			filename)
	}
	defer file.Close()

	gofluent10g.Log(gofluent10g.LOG_INFO, "Writing results to file '%s'",
		filename)

	// iterate over all packet sizes
	for i, pktlen := range pktlens {

		gofluent10g.Log(gofluent10g.LOG_INFO,
			"%d/%d: Datarate: 4x %.2f bps, Packet Length: %d",
			i+1, len(pktlens), datarate, pktlen)

		gofluent10g.LogIncrementIndentLevel()
		gofluent10g.Log(gofluent10g.LOG_INFO, "Generating trace ...")

		// generate CBR traffic trace
		trace := utils.GenTraceCBR(datarate, pktlen, pktlen-4, duration, 1)

		// calculate required memory bandwidth for concurrent replay and capture
		// on all four network interfaces
		// --> 4 network interfaces: 4x
		// --> concurrent replay and capture: 2x
		// --> reading + writing from DRAM: 2x
		// ----> 16x
		memBandwidth := 16.0 * 8.0 * float64(trace.GetSize()) /
			trace.GetDuration().Seconds()

		// write limit to output file
		file.WriteString(fmt.Sprintf("%d %f\n", pktlen, memBandwidth))

		// free host memory we do not need anymore
		trace = nil
		runtime.GC()

		gofluent10g.LogDecrementIndentLevel()
	}
}
