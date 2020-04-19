package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	rpio "github.com/stianeikeland/go-rpio"
)

const (
	hysteresis     = 0.1
	timeInterval   = 2 * time.Minute
	sensorRetry    = 3 * time.Second
	gettempBinary  = "Local/gettemp"
	ON             = "\033[1;32mON\033[0m"
	OFF            = "\033[1;31mOFF\033[0m"
	tempFormatter  = "\033[1;33m%.2f\033[0m"
	errorFormatter = "\033[1;31m%v\033[0m"
)

var (
	powerOn      = true // Weather we are powering the heater
	powerReading = true // Weather the heater has power (could be powered externally)
	thermostatOn = true // Whether the thermostat control is on
	heatOn       = true // Whether we are intending to connect the heat
	heatReading  = true // Whether the heat is actually connected (could be through an external thermostat)
	sensor       = "salon"
	currentTemp  = 0.0
	targetTemp   = 21.0
	errorInTemp  = true
	logfileName  = ""
	powerPin1    = rpio.Pin(14)
	powerPin2    = rpio.Pin(15)
	heatPin      = rpio.Pin(23)
	readPowerPin = rpio.Pin(27)
	readHeatPin  = rpio.Pin(17)
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func setPower(state string) {
	if state == ON {
		powerPin1.Write(rpio.High)
		powerPin2.Write(rpio.High)
		powerOn = true
	} else if state == OFF {
		powerPin1.Write(rpio.Low)
		powerPin2.Write(rpio.Low)
		powerOn = false
	}
	log.Println("Power set to", state)
}

func setHeat(state string) {
	if state == ON {
		heatPin.Write(rpio.High)
		heatOn = true
	} else if state == OFF {
		heatPin.Write(rpio.Low)
		heatOn = false
	}
	log.Println("Heat set to", state)
}

func readPower() bool {
	powerReading = readPowerPin.Read() == rpio.Low
	return powerReading
}

func readHeat() bool {
	heatReading = readHeatPin.Read() == rpio.High
	return heatReading
}

func readTemp() (temperature float64, err error) {
	cmd := exec.Command("ssh", "pi@"+sensor, gettempBinary)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		errorInTemp = true
		return
	}
	result := strings.TrimSpace(string(stdout.Bytes()))
	temperature, err = strconv.ParseFloat(result, 64)
	if err != nil {
		errorInTemp = true
		log.Printf("Error mneasuring temperature in sensor %v (%v)\n", sensor, err)
	} else {
		errorInTemp = false
		currentTemp = temperature
	}
	return
}

func printStatus() {
	readPower()
	readHeat()
	readTemp()
	fmt.Print("# Power should be ")
	if powerOn {
		fmt.Print(ON)
	} else {
		fmt.Print(OFF)
	}
	fmt.Print(" (and is ")
	if powerReading {
		fmt.Println(ON, ")")
	} else {
		fmt.Println(OFF, ")")
	}

	if errorInTemp {
		fmt.Printf(errorFormatter, "# Error reading current temperature, reference sensor is "+sensor+"\n")
	} else {
		fmt.Printf("# Current temperature is "+tempFormatter+" (reference sensor is %v)\n", currentTemp, sensor)
	}

	if powerOn {
		fmt.Print("# Thermostat control is ")
		if !thermostatOn {
			fmt.Println(OFF)
		} else {
			fmt.Println(ON)
		}
		fmt.Printf("# Target temperature is "+tempFormatter+"\n", targetTemp)
		fmt.Print("# Heat should be ")
		if heatOn {
			fmt.Print(ON)
		} else {
			fmt.Print(OFF)
		}
		fmt.Print(" (and is ")
		if heatReading {
			fmt.Println(ON, ")")
		} else {
			fmt.Println(OFF, ")")
		}
	}
}

func main() {

	executable, err := os.Executable()
	check(err)
	logfileName = executable + ".log"

	logfile, err := os.OpenFile(logfileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	log.SetOutput(logfile)
	log.Println("Starting thermostat and all")
	defer logfile.Close()

	log.Println("Configuring rpio ...")
	check(rpio.Open())
	powerPin1.Output()
	powerPin2.Output()
	defer powerPin1.Input()
	defer powerPin2.Input()
	setPower(ON)
	heatPin.Output()
	setHeat(OFF)
	defer heatPin.Input()
	readPowerPin.Input()
	readPowerPin.PullUp()
	readHeatPin.Input()
	readHeatPin.PullUp()
	log.Println("Done configuring rpio ...")

	// Thermostat loop
	go func() {
		lastRetry := sensorRetry
		for {
			readPower()
			readHeat()
			readTemp()
			if errorInTemp {
				// Oops, there has been an error measuring the temperature
				setHeat(OFF)
				time.Sleep(lastRetry)
				lastRetry *= 2 // So we increase the wait time progressively in cummulative errors
				continue
			} else {
				lastRetry = sensorRetry
			}
			log.Println("Current temp is ", currentTemp)
			if powerReading && thermostatOn {
				if currentTemp <= targetTemp-hysteresis && !heatReading {
					setHeat(ON)
				} else if currentTemp >= targetTemp+hysteresis && heatReading {
					setHeat(OFF)
				}
			}
			time.Sleep(timeInterval)
		}
	}()

	time.Sleep(2 * time.Second) // This just to let time to the thermostat loop to read the initial value of the temperature
	printStatus()
	fmt.Println()

	// Console loop
	for {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("\nControl console: ")

		s, _ := reader.ReadString('\n')
		command := strings.Fields(s)
		str := ""
		if len(command) >= 1 {
			switch command[0] {
			case "exit":
				fmt.Println("Have a nice day!")
				log.Println("Ending program, closing log\n\n")
				os.Exit(0)
			case "status":
				printStatus()
			case "changeTemp":
				if len(command) != 2 {
					fmt.Println("Missing target temperature, syntax is: changeTemp <temp>")
					continue
				}
				oldTemp := targetTemp
				newTemp, err := strconv.ParseFloat(command[1], 64)
				if err != nil {
					fmt.Println("Wrong target temperature: ", command[1])
					continue
				}
				targetTemp = newTemp
				str = fmt.Sprintf("Target temperature changed, old temparture was %.2f, new temperature is %.2f", oldTemp, targetTemp)
			case "changeSensor":
				if len(command) != 2 {
					fmt.Println("Missing new sensor, syntax is: changeSensor <sensor>")
					continue
				}
				oldSensor := sensor
				sensor = command[1]
				str = "Sensor changed, old sensor was " + oldSensor + ", new sensor is " + command[1]
				readTemp()
			case "pauseThermostat":
				thermostatOn = false
				setHeat(OFF)
				str = "Thermostat function now paused (and heat stopped)"
			case "resumeThermostat":
				thermostatOn = true
				fmt.Printf("Thermostat function resumed\n")
				str = "Thermostat function now resumed"
			case "heaterOff":
				setHeat(OFF)
				str = "Heat manually disconnected"
			case "heaterOn":
				setHeat(ON)
				str = "Heat manually connected"
			case "powerOff":
				setPower(OFF)
				str = "Power manually disconnected"
			case "powerOn":
				setPower(ON)
				str = "Power manually connected"
			case "help":
				fmt.Println("COMMANDS:")
				fmt.Println("status - prints current status")
				fmt.Println("changeTemp <temp> - sets a new target temperature, e.g. 21.5")
				fmt.Println("changeSensor <sensor> - sets a new refernce temperature sensor, e.g. \"salon\"")
				fmt.Println("pauseThermostat - disables the thermostat function (also manually stops the heater)")
				fmt.Println("resumeThermostat - enables the thermostat function")
				fmt.Println("heaterOff - manually disconnects the heater (irrespective of the thermostat function)")
				fmt.Println("heaterOn - manually connects the heater (irrespective of the thermostat function)")
				fmt.Println("powerOff - manually disconnects the power")
				fmt.Println("powerOn - manually connects the power")
				fmt.Println("help - prints help ;-)")
				fmt.Println("exit - exists program")
			default:
				fmt.Printf("Unknown command %v\n", command)
			}
			fmt.Println(str)
			log.Println(str)
		}
	}

}
