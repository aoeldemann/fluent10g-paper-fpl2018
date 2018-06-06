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
// Simple DPDK applications that stores the inter-packet arrival times between
// PTP IEEE1588 packets to an output text file. The application has been
// created and tested with the Intel X710 Network Interface Card (3.2 ns
// timestamp resolution) [1].
//
// The Intel X710 NIC has a total of four IEEE1588 RX timestamp registers.
// Whenever an IEEE1588 packet is received, the NIC stores the arrival timestamp
// in one of the four registers. The register is then locked (i.e. the value is
// not overwritten) until it is read by software. If all four registers are
// occupied, no more timestamps are taken.
//
// This application expects traffic that includes bursts of four IEEE1588
// packets (if an IEEE1588 packet is received, it MUST be followed by three
// more). The application reads the timestamp values from the RX timestamp
// registers and stores the inter-packet arrival timestamps (i.e. difference
// between two RX timestamp values) in memory. When the application is aborted
// (Ctrl-C), inter-packet arrival timestamps are written to an output file
// 'timestamp_diffs_measured.dat' (one timestamp per line).
//
// The application must be executed on a single lcore with a single assigned
// ethernet device.
//
// [1]
// https://www.intel.com/content/dam/www/public/us/en/documents/datasheets/xl710-10-40-controller-datasheet.pdf
//

#include <rte_cycles.h>
#include <rte_ethdev.h>
#include <rte_log.h>
#include <rte_malloc.h>
#include <signal.h>

#define NB_MEMPOOL 8192 // memory pool size
#define NB_RX_DESC 256  // # of entries in rx descriptor queue
#define NB_RX_BURST 32  // rx burst size

#define NB_TIMESTAMP_DIFFS                                                     \
  10 * 1000 * 1000 // maximum number of timestamped packets

// abort flag
uint8_t force_quit = 0;

// list of four suceeding mbufs that contain PTP packets, which hardware
// timestamped
struct rte_mbuf *mbufs_ts[4];

// number of valid mbufs in the mbufs_ts list
uint8_t nb_mbufs_ts = 0;

// ethernet device configuration
const struct rte_eth_conf port_conf = {.rxmode =
                                           {
                                               .split_hdr_size = 0,
                                               .header_split = 0,
                                               .hw_ip_checksum = 0,
                                               .hw_vlan_filter = 0,
                                               .jumbo_frame = 0,
                                               .hw_strip_crc = 0,
                                           },
                                       .txmode =
                                           {
                                               .mq_mode = ETH_MQ_TX_NONE,
                                           }};

// total number of received packets
uint64_t nb_pkts_rx = 0;

// total number of received timestamped packets
uint64_t nb_pkts_rx_ptp = 0;

// timestamp diff storage and current write position
uint8_t *ts_diff_mem;
uint32_t ts_diff_mem_wr = 0;

static void force_quit_handler(int32_t sig_num) {
  // set flag high when abort interrupt was triggered
  if (sig_num == SIGINT || sig_num == SIGTERM) {
    force_quit = 1;
  }
  printf("\n");
}

static void timespec_diff(struct timespec *start, struct timespec *end,
                          struct timespec *diff) {
  if ((end->tv_nsec - start->tv_nsec) >= 0) {
    diff->tv_sec = end->tv_sec - start->tv_sec;
    diff->tv_nsec = end->tv_nsec - start->tv_nsec;
  } else {
    diff->tv_sec = end->tv_sec - start->tv_sec - 1;
    diff->tv_nsec = end->tv_nsec - start->tv_nsec + 1000000000;
  }
}

static void save_timestamp_diff(uint64_t tv_nsec) {
  *((uint64_t *)(ts_diff_mem + ts_diff_mem_wr)) = tv_nsec;
  ts_diff_mem_wr += sizeof(uint64_t);
}

static void write_timestamp_diffs(const char *fname) {
  uint32_t pos = 0;
  uint64_t tv_nsec;

  // open file for writing
  FILE *f = fopen(fname, "w");

  while (pos < ts_diff_mem_wr) {
    tv_nsec = *((uint64_t *)(ts_diff_mem + pos));
    pos += sizeof(uint64_t);

    // write to file
    fprintf(f, "%lu\n", tv_nsec);
  }

  // close file
  fclose(f);
}

static void eval_inter_packet_times(void) {
  uint8_t i;
  int32_t retval;
  struct timespec ts[4], ts_diff;

  // first read all four timestamps
  for (i = 0; i < 4; i++) {
    retval = rte_eth_timesync_read_rx_timestamp(0, &ts[i],
                                                mbufs_ts[i]->timesync & 0x3);
    if (retval < 0) {
      rte_exit(-1, "invalid timestamp\n");
    }
  }

  for (i = 0; i < 4; i++) {
    if (i > 0) {
      // calculate difference between timestamps
      timespec_diff(&ts[i - 1], &ts[i], &ts_diff);

      // save timestamp difference
      save_timestamp_diff(ts_diff.tv_nsec);
    }

    // don't need the mbuf anymore
    rte_pktmbuf_free(mbufs_ts[i]);

    nb_pkts_rx_ptp += 1;
  }
}

static int32_t lcore_main(__rte_unused void *arg) {
  struct rte_mbuf *mbufs[NB_RX_BURST];
  struct rte_mbuf *mbuf;
  uint8_t nb_pkts;
  uint8_t i;

  // ptp packets must always be come in bursts of four. when the first one is
  // received, the flag is set high to mark an active burst. as long as the
  // flag is set, the next packet must also be a ptp packet
  uint8_t ptp_burst_active = 0;

  while (!force_quit) { // loop and loop until abort is requested
    // receive burst of packets
    nb_pkts = rte_eth_rx_burst(0, 0, mbufs, NB_RX_BURST);
    if (nb_pkts == 0) {
      // no packets received
      continue;
    }

    nb_pkts_rx += nb_pkts;

    // iterate over all received packets
    for (i = 0; i < nb_pkts; i++) {
      // get pointer on current mbuf
      mbuf = mbufs[i];

      if (mbuf->ol_flags & PKT_RX_IEEE1588_TMST) {
        // hardware timestamped this packet!
        if (nb_mbufs_ts == 0) {
          // this packet starts a burst of four ptp packets
          ptp_burst_active = 1;
        }

        // hardware timestamped this packet! save a pointer for later use
        mbufs_ts[nb_mbufs_ts] = mbuf;
        nb_mbufs_ts++;

        // if we received four timestamped ptp packets in a row, we start
        // evaluation
        if (nb_mbufs_ts == 4) {
          eval_inter_packet_times();

          // do not need the mbufs of the timestamped packets anymore
          nb_mbufs_ts = 0;

          // burst over
          ptp_burst_active = 0;
        }
      } else {
        if (ptp_burst_active) {
          rte_exit(-1, "expected timestamped ptp packet, did not get one\n");
        }
        rte_pktmbuf_free(mbuf);
      }
    }
  }

  return 0;
}

static void wait_link_up(void) {
  struct rte_eth_link link;

  RTE_LOG(INFO, PORT, "waiting for link to come up\n");

  while (!force_quit) {
    memset(&link, 0, sizeof(link));
    rte_eth_link_get_nowait(0, &link);
    if (link.link_status) {
      RTE_LOG(INFO, PORT, "link up\n");
      return;
    }
    rte_delay_ms(100);
  }
}

int main(int argc, char **argv) {
  int32_t retval;

  // initialize rte eal
  retval = rte_eal_init(argc, argv);
  if (retval < 0) {
    rte_exit(-1, "failed to init eal\n");
  }

  // make sure the number of ethernet device is one
  if (rte_eth_dev_count() != 1) {
    rte_exit(-1, "# of eth devs != 1\n");
  }

  // make sure number of lcores is one
  if (rte_lcore_count() != 1) {
    rte_exit(-1, "# of lcores != 1\n");
  }

  // create mbuf pool
  struct rte_mempool *pktmbuf_pool =
      rte_pktmbuf_pool_create("mbuf_pool", NB_MEMPOOL, 32, 0,
                              RTE_MBUF_DEFAULT_BUF_SIZE, rte_socket_id());

  if (pktmbuf_pool == NULL) {
    rte_exit(-1, "could not create mbuf pool\n");
  }

  // init eth device with a single rx queue
  retval = rte_eth_dev_configure(0, 1, 0, &port_conf);
  if (retval < 0) {
    rte_exit(-1, "could not configure eth dev\n");
  }

  // set mtu to 1520
  retval = rte_eth_dev_set_mtu(0, 1520);
  if (retval < 0) {
    rte_exit(-1, "could not set mtu to 1520\n");
  }

  // initialize rx queue
  retval = rte_eth_rx_queue_setup(0, 0, NB_RX_DESC, rte_eth_dev_socket_id(0),
                                  NULL, pktmbuf_pool);
  if (retval < 0) {
    rte_exit(-1, "could not init rx queue\n");
  }

  // start eth device
  retval = rte_eth_dev_start(0);
  if (retval < 0) {
    rte_exit(-1, "could not start ethernet device\n");
  }

  // register interrupt signal handler
  signal(SIGINT, force_quit_handler);
  signal(SIGTERM, force_quit_handler);

  // wait for link to come up
  wait_link_up();

  // enable promiscuous mode
  rte_eth_promiscuous_enable(0);

  // enable ieee1588 timestamping
  retval = rte_eth_timesync_enable(0);
  if (retval < 0) {
    rte_exit(-1, "could not enable ieee1588 timestamping\n");
  }

  // reserve memory to store timestamp difference in
  ts_diff_mem = rte_malloc(NULL, NB_TIMESTAMP_DIFFS * sizeof(uint64_t), 0);

  // start lcore thread on master core
  lcore_main(NULL);

  // disable ieee1588 timestamping
  rte_eth_timesync_disable(0);

  // stop and close eth dev
  rte_eth_dev_stop(0);
  rte_eth_dev_close(0);

  RTE_LOG(INFO, USER1, "Total number of received packets: %lu\n", nb_pkts_rx);
  RTE_LOG(INFO, USER1, "Total number of evaluated PTP packets: %lu\n",
          nb_pkts_rx_ptp);

  // write recorded timestamp differences to file
  write_timestamp_diffs("timestamp_diffs_measured.dat");

  // free timestamp difference memory
  rte_free(ts_diff_mem);

  return 0;
}
