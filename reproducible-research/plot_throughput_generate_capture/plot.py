#!/usr/bin/env python
"""Plot measurement results."""
# The MIT License
#
# Copyright (c) 2017-2018 by the author(s)
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
#
# Author(s):
#   - Andreas Oeldemann <andreas.oeldemann@tum.de>
#
# Description:
#
# see README.md

import os
import sys
import matplotlib.pyplot as plt


def main(argv):
    """Main function."""
    # plotting data that the user measured or the data that we provide?
    if len(argv) > 1 and argv[1] == "-ref":
        dataDir = "output_ref/"
    else:
        dataDir = "output/"

    # assemble input file name
    fname = os.path.join(dataDir, "max_throughput.dat")

    # make sure input file exists
    if not os.path.isfile(fname):
        print("Input file 'max_throughput.dat' does not exist. Either perform")
        print("a measurement by running 'sudo go run main.go' or call this")
        print("script with the '-ref' command line option to plot our")
        print("measurement data located in the 'output_ref' directory.")
        return

    pktlens = []
    networkThroughputs = []
    memoryBandwidths = []

    # open result file for reading
    with open(fname, 'r') as f:
        # each line in the input file contains a tuple with the following
        # values seperated by a whitespace:
        #  0: packet length
        #  1: network throughput
        #  2: resulting memory bandwidth
        for line in f:
            lineSplit = line.split(' ')
            pktlens.append(int(lineSplit[0]))
            networkThroughputs.append(float(lineSplit[1]))
            memoryBandwidths.append(float(lineSplit[2]))

    # sort packet sizes and network throughputs based on packet sizes
    pktlens, networkThroughputs = (list(t) for t in
                                   zip(*sorted(zip(pktlens,
                                                   networkThroughputs))))

    # sort packet sizes and memory bandwidths based on packet sizes
    pktlens, memoryBandwidths = (list(t) for t in
                                 zip(*sorted(zip(pktlens,
                                                 memoryBandwidths))))

    # convert network throughputs and memory bandwidths to Gbps
    networkThroughputs = map(lambda x: x/1e9, networkThroughputs)
    memoryBandwidths = map(lambda x: x/1e9, memoryBandwidths)

    # create subplots
    fig, axs = plt.subplots(nrows=2, ncols=1)

    # plot network throughputs and memory bandwidths
    axs[0].plot(pktlens, networkThroughputs, marker='x')
    axs[1].plot(pktlens, memoryBandwidths, marker='x')

    # set axes limits
    axs[0].set_xlim(min(pktlens), max(pktlens))
    axs[1].set_xlim(min(pktlens), max(pktlens))

    # set axes labels
    axs[0].set_xlabel("Packet Size [byte]")
    axs[1].set_xlabel("Packet Size [byte]")
    axs[0].set_ylabel("Network Throughput (duplex) [Gbps]")
    axs[1].set_ylabel("Aggregate Memory Bandwidth [Gbps]")

    # show the plot
    plt.show()


if __name__ == "__main__":
    main(sys.argv)
