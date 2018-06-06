#!/usr/bin/env python
"""Plot results."""
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
    # plotting data that the user genereated or the data that we provide?
    if len(argv) > 1 and argv[1] == "-ref":
        dataDir = "output_ref/"
    else:
        dataDir = "output/"

    # assemble result filename
    fname = os.path.join(dataDir, "required_membandwidth.dat")

    # make sure that output file exists
    if not os.path.isfile(fname):
        print("No data has been found. Either generate data by running")
        print("'sudo go run main.go' or call this script with the '-ref'")
        print("command line option to plot data located in the 'output_ref'")
        print("directory.")
        return

    # open input file and read packet sizes and required memory bandwidth
    pktlens = []
    memoryBandwidths = []
    with open(fname, 'r') as f:
        for line in f:
            lineSplit = line.split(' ')
            pktlen = int(lineSplit[0])
            memoryBandwidth = float(lineSplit[1])
            pktlens.append(pktlen)
            memoryBandwidths.append(memoryBandwidth)

    # sort both lists based on pktlen
    pktlens, memoryBandwidths = (list(t) for t in
                                 zip(*sorted(zip(pktlens, memoryBandwidths))))

    # convert bandwidth to Gbps
    memoryBandwidths = map(lambda x: x/1e9, memoryBandwidths)

    # plot data
    plt.plot(pktlens, memoryBandwidths, marker='x')

    # set axes limits
    plt.xlim([min(pktlens), max(pktlens)])
    plt.ylim([min(memoryBandwidths), max(memoryBandwidths)])

    # label axes
    plt.xlabel("Packet Size [byte]")
    plt.ylabel("Required Memory Bandwidth [Gbps]")

    # show grid
    plt.grid()

    # show plot
    plt.show()


if __name__ == "__main__":
    main(sys.argv)
