// Copyright 2022 - Sander Tolsma. All rights reserved
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/stolsma/go-p4pack/pkg/cli"
	"github.com/stolsma/go-p4pack/pkg/dpdkinfra"
	"github.com/stolsma/go-p4pack/pkg/dpdkswx/ethdev"
	"github.com/stolsma/go-p4pack/pkg/dpdkswx/tap"
)

func InterfaceCreateCmd(parents ...*cobra.Command) *cobra.Command {
	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Base command for all interface create actions",
		Aliases: []string{"cr"},
	}

	InterfaceCreateTapCmd(createCmd)
	InterfaceCreateEthdevCmd(createCmd)
	return cli.AddCommand(parents, createCmd)
}

func InterfaceCreateTapCmd(parents ...*cobra.Command) *cobra.Command {
	tapCmd := &cobra.Command{
		Use:   "tap [name] [pktmbuf] [mtu]",
		Short: "Create a tap interface on the system",
		Args:  cobra.ExactArgs(3),
		ValidArgsFunction: cli.ValidateArguments(
			cli.AppendHelp("You must choose a name for the tap interface you are adding"),
			completePktmbufArg,
			cli.AppendHelp("You must specify the MTU for the tap interface you are adding"),
			cli.AppendLastHelp(3, "This command does not take any more arguments"),
		),
		Run: func(cmd *cobra.Command, args []string) {
			dpdki := dpdkinfra.Get()
			var params tap.Params

			// get pktmbuf
			arg1 := args[1]
			params.Pktmbuf = dpdki.PktmbufStore.Get(arg1)
			if params.Pktmbuf == nil {
				cmd.PrintErrf("Pktmbuf %s not defined!\n", arg1)
				return
			}

			// get MTU
			mtu, err := strconv.ParseInt(args[2], 0, 16)
			if err != nil {
				cmd.PrintErrf("Mtu (%s) is not a correct integer: %d\n", args[2], err)
				return
			}
			params.Mtu = int(mtu)

			// create
			_, err = dpdki.TapCreate(args[0], &params)
			if err != nil {
				cmd.PrintErrf("TAP %s create err: %d\n", args[0], err)
				return
			}

			cmd.Printf("TAP %s created!\n", args[0])
		},
	}

	return cli.AddCommand(parents, tapCmd)
}

func InterfaceCreateEthdevCmd(parents ...*cobra.Command) *cobra.Command {
	ethdevCmd := &cobra.Command{
		Use:   "ethdev [name] [device] [pktmbuf] [# tx queues] [tx queuesize] [# rx queues] [rx queuesize] [mtu] [promiscuous]",
		Short: "Create an ethdev interface on the system",
		Args:  cobra.MatchAll(cobra.MinimumNArgs(7), cobra.MaximumNArgs(9)),
		ValidArgsFunction: cli.ValidateArguments(
			cli.AppendHelp("You must choose a name for the ethdev interface you are adding"),
			completeUnusedEthdevPortList,
			completePktmbufArg,
			cli.AppendHelp("You must specify the number of transmit queues for the ethdev interface you are adding"),
			cli.AppendHelp("You must specify the transmit queuesize for the ethdev interface you are adding"),
			cli.AppendHelp("You must specify the number of receive queues for the ethdev interface you are adding"),
			cli.AppendHelp("You must specify the receive queuesize for the ethdev interface you are adding"),
			cli.AppendHelp("You must specify the MTU for the ethdev interface you are adding (0/none is default value of interface)"),
			cli.AppendHelp("You must specify the promiscuous mode for the tap interface you are adding (on/off, on is default)"),
			cli.AppendLastHelp(9, "This command does not take any more arguments"),
		),
		Run: func(cmd *cobra.Command, args []string) {
			dpdki := dpdkinfra.Get()
			var params ethdev.Params

			// get device name
			params.PortName = args[1]

			// get pktmbuf
			params.Rx.Mempool = dpdki.PktmbufStore.Get(args[2])
			if params.Rx.Mempool == nil {
				cmd.PrintErrf("Pktmbuf %s not defined!\n", args[2])
				return
			}

			// get # TX Queues
			ntxq, err := strconv.ParseInt(args[3], 0, 16)
			if err != nil {
				cmd.PrintErrf("# TX Queues (%s) is not a correct integer: %v\n", args[3], err)
				return
			}
			params.Tx.NQueues = uint16(ntxq)

			// get TX Queuesize
			txqsize, err := strconv.ParseInt(args[4], 0, 32)
			if err != nil {
				cmd.PrintErrf("TX Queuesize (%s) is not a correct integer: %v\n", args[4], err)
				return
			}
			params.Tx.QueueSize = uint32(txqsize)

			// get # RX Queues
			nrxq, err := strconv.ParseInt(args[5], 0, 16)
			if err != nil {
				cmd.PrintErrf("# RX Queues (%s) is not a correct integer: %v\n", args[5], err)
				return
			}
			params.Rx.NQueues = uint16(nrxq)

			// get RX Queuesize
			rxqsize, err := strconv.ParseInt(args[6], 0, 32)
			if err != nil {
				cmd.PrintErrf("RX Queuesize (%s) is not a correct integer: %v\n", args[6], err)
				return
			}
			params.Rx.QueueSize = uint32(rxqsize)

			// get MTU if available
			var mtu int64 = 1500
			if len(args) > 7 {
				mtu, err = strconv.ParseInt(args[7], 0, 16)
				if err != nil {
					cmd.PrintErrf("MTU (%s) is not a correct integer: %v\n", args[7], err)
					return
				}
				if mtu == 0 {
					mtu = 1500
				}
			}
			params.Rx.Mtu = uint16(mtu)

			// get promiscuous mode if available, else default value
			var prom = true
			if len(args) > 8 {
				val := strings.ToLower(args[8])
				switch val {
				case "on":
					prom = true
				case "off":
					prom = false
				default:
					cmd.PrintErrf("Promiscuous mode value (%s) should be `on` or `off`\n", args[8])
					return
				}
			}
			params.Promiscuous = prom

			// TODO Rx RSS needs to be implemented!
			//	p.Rx.Rss = vh.Rx.Rss

			// create
			_, err = dpdki.EthdevCreate(args[0], &params)
			if err != nil {
				cmd.PrintErrf("Ethdev %s create err: %v\n", args[0], err)
				return
			}

			cmd.Printf("Ethdev %s created!\n", args[0])
		},
	}

	return cli.AddCommand(parents, ethdevCmd)
}
