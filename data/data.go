package data

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stianeikeland/go-rpio"
)

const (
	configFileName     = ".calderaConfig"
	gettempBinary      = "Local/gettemp"
	ON                 = "\033[1;32mON\033[0m"
	OFF                = "\033[1;31mOFF\033[0m"
	OilWarning         = 1000
	OilCriticalWarning = 600
	MinTemp            = 1 // If temperature is less than this then we consider the temp sensor is not working properly
)

var (
	PowerOn         = true // Weather we are powering the heater
	PowerReading    = true // Weather the heater has power (could be powered externally)
	ThermostatOn    = true // Whether the thermostat control is on
	HeatOn          = true // Whether we are intending to connect the heat
	HeatReading     = true // Whether the heat is actually connected (could be through an external thermostat)
	Sensor          = "salon"
	CurrentTemp     = 0.0
	TargetTemp      = 21.0
	Hysteresis      = 0.05
	ErrorInTemp     = true
	LogfileName     = ""
	PowerPin1       = rpio.Pin(14)
	PowerPin2       = rpio.Pin(15)
	HeatPin         = rpio.Pin(23)
	ReadPowerPin    = rpio.Pin(27)
	ReadHeatPin     = rpio.Pin(17)
	LastOilRead     = 0
	LastOilReadDate = time.Unix(0, 0)
	LastConsumption = 0.0
	M               = sync.Mutex{}
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
	//result := strings.TrimSpace(string(stdout.Bytes()))
	result := strings.TrimSpace(stdout.String())
	temperature, err = strconv.ParseFloat(result, 64)
	if err != nil {
		ErrorInTemp = true
		log.Printf("Error measuring temperature in sensor %v (%v)\n", Sensor, err)
	} else if temperature < MinTemp {
		ErrorInTemp = true
		errMsg := fmt.Sprintf("Error measuring temperature in sensor %v, temp is %v and that seems too low (min threshold is %v)", Sensor, temperature, MinTemp)
		err = errors.New(errMsg)
		log.Print(errMsg)
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

func ReadConfig() {
	configFile, err := os.Open(configFileName)
	defer configFile.Close()
	if err == nil {

		var powerOnSaved bool
		var thermostatOnSaved bool
		var heatOnSaved bool
		var sensorSaved string
		var targetTempSaved float64
		var hysteresisSaved float64

		var line string
		n, err := fmt.Fscanln(configFile, &line)
		if err != nil || n != 1 {
			return
		}
		s := strings.Split(line, ",")
		if len(s) != 6 {
			return
		}

		if s[0] == "true" {
			powerOnSaved = true
		} else if s[0] == "false" {
			powerOnSaved = false
		} else {
			return
		}

		if s[1] == "true" {
			thermostatOnSaved = true
		} else if s[1] == "false" {
			thermostatOnSaved = false
		} else {
			return
		}

		if s[2] == "true" {
			heatOnSaved = true
		} else if s[2] == "false" {
			heatOnSaved = false
		} else {
			return
		}

		sensorSaved = s[3]

		targetTempSaved, err = strconv.ParseFloat(s[4], 64)
		if err != nil {
			return
		}

		hysteresisSaved, err = strconv.ParseFloat(s[5], 64)
		if err != nil {
			return
		}

		PowerOn = powerOnSaved
		ThermostatOn = thermostatOnSaved
		HeatOn = heatOnSaved
		Sensor = sensorSaved
		TargetTemp = targetTempSaved
		Hysteresis = hysteresisSaved

	} else {
		fmt.Println("Config file does not exist")
	}
}

func WriteConfig() {
	configFile, err := os.Create(configFileName)
	if err == nil {
		fmt.Fprintf(configFile, "%v,%v,%v,%v,%v,%v\n", PowerOn, ThermostatOn, HeatOn, Sensor, TargetTemp, Hysteresis)
	}
	configFile.Close()
	log.Println("Config updated:")
	log.Println("  - powerOn is", PowerOn)
	log.Println("  - heatOn is", HeatOn)
	log.Println("  - thermostatOn", ThermostatOn)
	log.Println("  - sensor is", Sensor)
	log.Println("  - targetTemp is", TargetTemp)
	log.Println("  - hysteresis is", Hysteresis)
}
