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
import re
import sys
import matplotlib.pyplot as plt
import numpy as np

T_CLK = 6.4  # ns


def main(argv):
    """Main function."""
    # plotting data that the user measured or the data that we provide?
    if len(argv) > 1 and argv[1] == "-ref":
        dataDir = "output_ref/"
    else:
        dataDir = "output/"

    # find all histogram data files
    histfiles = []
    for fname in os.listdir(dataDir):
        if fname.endswith(".dat"):
            # extract datarate in bps from filename
            m = re.match("^histogram_(.*?).dat$", fname)

            # skip this file if it did not match the pattern we are expecting
            if not m:
                continue

            # calculate datarate in Gbps
            datarateMean = float(m.group(1)) / 1e9

            # add file to list
            histfiles.append((datarateMean, os.path.join(dataDir, fname)))

    # abort if we did not find any files
    if len(histfiles) == 0:
        print("No measurement data has been found. Either perform a")
        print("measurement by running 'sudo go run main.go' or call this")
        print("script with the '-ref' command line option to plot out")
        print("measurement data located in the 'output_ref' directory.")
        return

    # sort file list in ascending mean data rate order
    histfiles.sort(key=lambda x: x[0])

    # create subplots
    fig, axs = plt.subplots(nrows=len(histfiles), ncols=1, sharex=True)

    # iterate over files
    for i, histfile in enumerate(histfiles):
        # open file for reading
        f = open(histfile[1], 'r')

        # we will fill two list: one containing the histogram bin latency
        # value, one containing the number of occurrences of each latency
        # value
        latencies = []
        occurrences = []

        # each line in the input file list containts a tuple of of latency
        # bin value and number of occurrences (seperated by a whitespace)
        for line in f:
            lineSplit = line.split(' ')
            latencies.append(float(lineSplit[0]))
            occurrences.append(float(lineSplit[1]))

        # calculate the probability of each latency value in percent
        probabilities = map(lambda x: 100.0 * float(x)/sum(occurrences),
                            occurrences)

        # record min and max latency values
        if i == 0:
            latency_min = min(latencies)
        elif min(latencies) < latency_min:
            latency_min = min(latencies)
        if i == 0:
            latency_max = max(latencies)
        elif max(latencies) > latency_max:
            latency_max = max(latencies)

        # plot histogram
        bars = axs[i].bar(latencies, probabilities, align="center",
                          width=T_CLK/1.5,
                          label="Mean datarate: %.2lf Gbps" % histfile[0])

        # show probability values over bars
        for bar in bars:
            height = bar.get_height()
            axs[i].text(bar.get_x() + bar.get_width()/2.0, 1.05*height,
                        '%.6lf %%' % height, ha="center", va="bottom")

        # add legend
        axs[i].legend()

    # configure subplots
    for i, ax in enumerate(axs):
        ax.set_ylim([0, 110])
        ax.set_xlim([latency_min - 1.0, latency_max + 1.0])
        ax.xaxis.set_ticks(np.arange(latency_min, latency_max + T_CLK, T_CLK))
        ax.yaxis.set_ticks(np.arange(0, 110, 10))
        ax.grid()

        if i != len(axs)-1:
            plt.setp(ax.get_xticklabels(), visible=False)

    # no horizontal spacing between plots
    fig.subplots_adjust(hspace=0)

    # shared label for x and y axis
    fig.text(0.5, 0.05, "Measured Latency [ns]", ha='center')
    fig.text(0.05, 0.5, "Probability [%]", va="center", rotation="vertical")

    # show the plot
    plt.show()


if __name__ == "__main__":
    main(sys.argv)
