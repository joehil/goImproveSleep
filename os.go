//go:build !baremetal

package main

import "os"

func connectAddress() (string, string) {
        if len(os.Args) < 2 {
                println("usage: heartrate-monitor [address] [interval]")
                os.Exit(1)
        }

        // look for device with specific name
        address := os.Args[1]
        interval := string("9")

        if len(os.Args) == 3 {
                interval = os.Args[2]
        }

        return address, interval
}

// done just prints a message and allows program to exit.
func done() {
        println("Done.")
}
