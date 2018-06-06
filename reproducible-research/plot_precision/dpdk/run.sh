#!/bin/bash
#
# Run on a single core (e.g. core mask -c 0x1).
# Run with a single network interface bound to a dpdk driver.
#

sudo build/fluent10g_precision $@
