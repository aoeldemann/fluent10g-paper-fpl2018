# Reproducible Research

The code in the subfolders was used to obtain the evaluation results presented
in our FPL 2018 paper. By providing the code we enable interested users to
independently verify our performance/precision/accuracy results.

## Measurement Setup

Our measurements were performed with the following hardware setup:

* NetFPGA-SUME FPGA board
* Dell PowerEdge T630 workstation, 1x Intel Xeon E5-2620 v3 CPU @ 2.40 Ghz, 4x M393A1G43DB0-CPB DDR4-2133MHz memory (total of 32 GByte)
* 4x Avago AFBR-703SDZ optical transceivers (for performance and accuracy measurements)
* 2x Amphenol FO-10GGBLCX20-001 1m multimode OM3 fibers (for performance and accuracy measurements)
* Intel X710-DA2 10 GbE network interface card (firmware version 6.01, for precision measurements)
* 1x 3m Intel XDACBL3M direct-attach SFP+ copper cable (for precision measurements)

Our software setup:

* Ubuntu 16.04 Server, Linux Kernel version 4.4.0-127
* [Xilinx XDMA driver](https://www.xilinx.com/support/answers/65444.html)
    (poll-mode)
* Go version 1.6.2

## Git Commits

The measurements have been performed with the code base of the following Git
commits:

* fluent10g: `b11bc76bdf64c612e6f806a71125c10e1b196aa2`
* gofluent10g: `8ab4e0cbe5970bccd07c8f2482b8abcdb5fdda74`
