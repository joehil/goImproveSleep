package main

import (
	"container/list"
	"fmt"
	"math"
	"strconv"
	"time"

	"tinygo.org/x/bluetooth"
)

var (
	adapter = bluetooth.DefaultAdapter

	heartRateServiceUUID        = bluetooth.ServiceUUIDHeartRate
	heartRateCharacteristicUUID = bluetooth.CharacteristicUUIDHeartRateMeasurement
	heartService2UUID           = bluetooth.UUID([]uint32{0, 0, 0, 0})
	heartCharacteristic2UUID    = bluetooth.UUID([]uint32{0, 0, 0, 0})
)

var rrs = list.New()
var hrvAddress, interval string
var intInterval int

func main() {
	println("enabling")

	heartService2UUID, _ := bluetooth.ParseUUID("6e400001-b5a3-f393-e0a9-e50e24dcca9e")
	heartCharacteristic2UUID, _ := bluetooth.ParseUUID("6e400003-b5a3-f393-e0a9-e50e24dcca9e")

	// Enable BLE interface.
	must("enable BLE stack", adapter.Enable())

	ch := make(chan bluetooth.ScanResult, 1)

	// Start scanning.
	println("scanning...")
	err := adapter.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		println("found device:", result.Address.String(), result.RSSI, result.LocalName())
		hrvAddress, interval = connectAddress()
		if result.Address.String() == hrvAddress {
			adapter.StopScan()
			ch <- result
		}
	})

	var device *bluetooth.Device
	select {
	case result := <-ch:
		device, err = adapter.Connect(result.Address, bluetooth.ConnectionParams{})
		if err != nil {
			println(err.Error())
			return
		}

		println("connected to ", result.Address.String())
	}

	intInterval, _ = strconv.Atoi(interval)
	fmt.Printf("Interval: %d\n", intInterval)
	// get services
	println("discovering services/characteristics")
	srvcs, err := device.DiscoverServices(nil)
	fmt.Println(srvcs)

	fmt.Println([]bluetooth.UUID{heartRateServiceUUID, bluetooth.ServiceUUIDSecureDFU})

	srvcs, err = device.DiscoverServices([]bluetooth.UUID{heartRateServiceUUID})
	if srvcs == nil {
		srvcs, err = device.DiscoverServices([]bluetooth.UUID{heartService2UUID})
	}

	//	must("discover services", err)
	fmt.Println(srvcs)

	if len(srvcs) == 0 {
		panic("could not find heart rate service")
	}

	srvc := srvcs[0]

	println("found service", srvc.UUID().String())

	chars, err := srvc.DiscoverCharacteristics(nil)
	fmt.Println(chars)

	chars, err = srvc.DiscoverCharacteristics([]bluetooth.UUID{heartRateCharacteristicUUID})
	if chars == nil {
		chars, err = srvc.DiscoverCharacteristics([]bluetooth.UUID{heartCharacteristic2UUID})
	}
	if err != nil {
		println(err)
	}

	if len(chars) == 0 {
		panic("could not find heart rate characteristic")
	}

	char := chars[0]
	println("found characteristic", char.UUID().String())

	if srvc.UUID() != heartService2UUID {
		char.EnableNotifications(func(buf []byte) {
			lvalue := uint16(0)
			hvalue := uint16(0)
			var hstate string
			rr := float64(0)
			if len(buf) > 2 {
				lvalue = uint16(buf[2])
				hvalue = uint16(uint16(buf[3]) * 256)
				rr = float64(lvalue+hvalue) * 1000 / 1024
				if rr > 1 {
					rrs.PushBack(rr)
				}
			}
			if uint8(buf[1]) < 58 {
				hstate = "asleep"
			} else {
				hstate = ""
			}
			fmt.Print(time.Now().Format(time.RFC850))
			fmt.Printf(" HR: %d RR: %0.0f HRV: %0.0f %s\n", uint8(buf[1]), rr, get_hrv(), hstate)
			//	fmt.Printf("%b\n", uint8(buf[0]))
		})
	} else {
		char.EnableNotifications(func(buf []byte) {
			if buf[4] == 1 {
				fmt.Println(buf)
				fmt.Printf("HR: %d PI: %0.1f SpO2: %d\n", uint8(buf[6]), float64(buf[8])/float64(10), uint8(buf[5]))
			} else {
				fmt.Println(buf)
			}
		})
	}

	select {}
}

func must(action string, err error) {
	if err != nil {
		panic("failed to " + action + ": " + err.Error())
	}
}

func get_hrv() float64 {
	var i int = 0
	lastv := float64(0)
	diff := float64(0)
	sum := float64(0)
	for e := rrs.Front(); e != nil; e = e.Next() {
		//		fmt.Printf("%0.0f  ", e.Value)
		if i > 0 {
			diff = e.Value.(float64) - lastv
			sum += math.Abs(diff)
		}
		i++
		lastv = e.Value.(float64)
	}
	if rrs.Len() > intInterval {
		rrs.Remove((rrs.Front()))
	}
	//	fmt.Println(i)
	return sum / float64(i-1)
}
