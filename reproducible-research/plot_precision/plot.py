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
import math
import matplotlib.pyplot as plt
import numpy as np

T_CLK_NIC = 3.2  # ns


def main(argv):
    """Main function."""
    # plotting data that the user measured or the data that we provide?
    if len(argv) > 1 and argv[1] == "-ref":
        dataDir = "output_ref/"
    else:
        dataDir = "output/"

    # assemble file names
    fnameExpected = os.path.join(dataDir, "timestamp_diffs_expected.dat")
    fnameMeasured = os.path.join(dataDir, "timestamp_diffs_measured.dat")

    if os.path.isfile(fnameExpected) is False or \
       os.path.isfile(fnameMeasured) is False:
        print("No measurement data has been found. Either perform a")
        print("measurement by running 'sudo go run main.go' or call this")
        print("script with the '-ref' command line option to plot out")
        print("measurement data located in the 'output_ref' directory.")
        return

    tExpectedList = []
    tMeasuredList = []

    # read expected inter-packet times
    with open(fnameExpected) as f:
        for line in f:
            tExpectedList.append(float(line) * 1e9)

    # read measured inter-packet times
    with open(fnameMeasured) as f:
        for line in f:
            timestamp = int(line)
            # round to nearest clock period
            timestamp /= T_CLK_NIC
            timestamp = round(round(timestamp)*T_CLK_NIC, 1)
            tMeasuredList.append(timestamp)

    # make sure we have as many measured timestamps as we do expect
    assert len(tExpectedList) == len(tMeasuredList)

    # calculate absolute error
    tErrorList = map(lambda i: tMeasuredList[i] - tExpectedList[i],
                     range(len(tExpectedList)))

    # the accuracy of our measurements is determined by the clock frquency
    # of the NIC timestamping logic. bin the reulsts
    start = T_CLK_NIC * math.floor(min(tErrorList)/T_CLK_NIC)
    end = T_CLK_NIC * math.ceil(max(tErrorList)/T_CLK_NIC)

    errors = []
    occurences = []

    c = start
    while c < end:
        errors.append(round(c, 1))
        occurences.append(len(filter(lambda x: c <= x < c + T_CLK_NIC,
                          tErrorList)))
        c += T_CLK_NIC

    assert sum(occurences) == len(tErrorList)

    # translate occurences to proabilites
    probabilities = map(lambda x: 100.0*float(x)/len(tErrorList), occurences)

    # create bar plot
    plt.bar(errors, probabilities, width=T_CLK_NIC)

    # configure plot
    plt.xlabel("Absolute Measured Inter-Packet Time Error [ns]")
    plt.ylabel("Probability [%]")
    plt.xticks(np.arange(min(errors), max(errors) + T_CLK_NIC, T_CLK_NIC))
    plt.ylim([0, 30])
    plt.grid()

    # show plot
    plt.show()


if __name__ == "__main__":
    main(sys.argv)
