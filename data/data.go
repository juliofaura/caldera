package data

import (
	"bytes"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/stianeikeland/go-rpio"
)

const (
	gettempBinary = "Local/gettemp"
	ON            = "\033[1;32mON\033[0m"
	OFF           = "\033[1;31mOFF\033[0m"
)

var (
	PowerOn      = true // Weather we are powering the heater
	PowerReading = true // Weather the heater has power (could be powered externally)
	ThermostatOn = true // Whether the thermostat control is on
	HeatOn       = true // Whether we are intending to connect the heat
	HeatReading  = true // Whether the heat is actually connected (could be through an external thermostat)
	Sensor       = "salon"
	CurrentTemp  = 0.0
	TargetTemp   = 21.0
	Hysteresis   = 0.05
	ErrorInTemp  = true
	LogfileName  = ""
	PowerPin1    = rpio.Pin(14)
	PowerPin2    = rpio.Pin(15)
	HeatPin      = rpio.Pin(23)
	ReadPowerPin = rpio.Pin(27)
	ReadHeatPin  = rpio.Pin(17)
)

func ReadPower() bool {
	PowerReading = ReadPowerPin.Read() == rpio.Low
	return PowerReading
}

func ReadHeat() bool {
	HeatReading = ReadHeatPin.Read() == rpio.High
	return HeatReading
}

func ReadTemp() (temperature float64, err error) {
	cmd := exec.Command("ssh", "pi@"+Sensor, gettempBinary)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		ErrorInTemp = true
		return
	}
	result := strings.TrimSpace(string(stdout.Bytes()))
	temperature, err = strconv.ParseFloat(result, 64)
	if err != nil {
		ErrorInTemp = true
		log.Printf("Error mneasuring temperature in sensor %v (%v)\n", Sensor, err)
	} else {
		ErrorInTemp = false
		CurrentTemp = temperature
	}
	return
}

func SetPower(state string) {
	if state == ON {
		PowerPin1.Write(rpio.High)
		PowerPin2.Write(rpio.High)
		PowerOn = true
	} else if state == OFF {
		PowerPin1.Write(rpio.Low)
		PowerPin2.Write(rpio.Low)
		PowerOn = false
	}
	log.Println("Power set to", state)
}

func SetHeat(state string) {
	if state == ON {
		HeatPin.Write(rpio.High)
		HeatOn = true
	} else if state == OFF {
		HeatPin.Write(rpio.Low)
		HeatOn = false
	}
	log.Println("Heat set to", state)
}
