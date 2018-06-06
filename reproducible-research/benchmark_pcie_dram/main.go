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
	"github.com/aoeldemann/gopcie"
	"sync"
	"time"
)

var (
	// read/write PCIExpress character devices
	PCIE_DEV_WR = "/dev/xdma0_h2c_0"
	PCIE_DEV_RD = "/dev/xdma0_c2h_0"

	// dma transfer size
	dmaTransferSize = 64 * 1024 * 1024

	// duration of the read and write benchmarks
	benchmarkDuration = 30 * time.Second

	// goroutine sync stuff
	syncWg   sync.WaitGroup
	syncChan chan bool
)

func rdwr(dev *gopcie.PCIeDMA, rdwr int, throughput *float64) {
	// rdwr == 0 => READ
	// rdwr == 1 => WRITE

	defer syncWg.Done()

	data := make([]byte, dmaTransferSize)

	var totalTime time.Duration
	var totalBytes uint64
	stop := false

	for {
		select {
		case _ = <-syncChan:
			stop = true
		default:
		}

		if stop {
			break
		}

		// record time before transfer
		transferStartTime := time.Now()

		// perform read / write
		if rdwr == 0 {
			dev.Read(0x0, data)
		} else {
			dev.Write(0x0, data)
		}

		// sum up transfer time and size
		totalTime += time.Since(transferStartTime)
		totalBytes += uint64(dmaTransferSize)
	}

	*throughput = 8.0 * float64(totalBytes) / totalTime.Seconds() / 1e9
}

func main() {
	// open devices
	devRd, err := gopcie.PCIeDMAOpen(PCIE_DEV_RD,
		gopcie.PCIE_ACCESS_READ)
	if err != nil {
		panic("could not open dev for reading")
	}
	defer devRd.Close()
	devWr, err := gopcie.PCIeDMAOpen(PCIE_DEV_WR,
		gopcie.PCIE_ACCESS_WRITE)
	if err != nil {
		panic("could not open dev for writing")
	}
	defer devWr.Close()

	// initialize sync channels
	syncChan = make(chan bool)

	var throughputRd, throughputWr float64

	// read benchmark
	syncWg.Add(1)
	go rdwr(devRd, 0, &throughputRd)
	time.Sleep(benchmarkDuration)
	syncChan <- true
	syncWg.Wait()

	// write benchmark
	syncWg.Add(1)
	go rdwr(devWr, 1, &throughputWr)
	time.Sleep(benchmarkDuration)
	syncChan <- true
	syncWg.Wait()

	// print out recorded throughput
	fmt.Printf("Read throughput: %.2f\n", throughputRd)
	fmt.Printf("Write throughput: %.2f\n", throughputWr)
	fmt.Printf("\n")
}
