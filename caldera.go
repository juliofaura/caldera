package main

//    To install this as a crontab:
//    */1 * * * * if [ ! $(pgrep caldera) ]; then tmux new-session -d -s auto-session /home/pi/Local/caldera; fi

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/juliofaura/caldera/data"
	"github.com/juliofaura/caldera/server"
	rpio "github.com/stianeikeland/go-rpio"
)

const (
	timeInterval   = 1 * time.Minute
	sensorRetry    = 3 * time.Second
	maxSensorRetry = 1 * time.Minute
	configFileName = ".calderaConfig"
	tempFormatter  = "\033[1;33m%.2f\033[0m"
	errorFormatter = "\033[1;31m%v\033[0m"
)

var ()

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func printStatus() {
	data.ReadPower()
	data.ReadHeat()
	data.ReadTemp()
	fmt.Print("# Power should be ")
	if data.PowerOn {
		fmt.Print(data.ON)
	} else {
		fmt.Print(data.OFF)
	}
	fmt.Print(" (and is ")
	if data.PowerReading {
		fmt.Println(data.ON, ")")
	} else {
		fmt.Println(data.OFF, ")")
	}

	if data.ErrorInTemp {
		fmt.Printf(errorFormatter, "# Error reading current temperature, reference sensor is "+data.Sensor+"\n")
	} else {
		fmt.Printf("# Current temperature is "+tempFormatter+" (reference sensor is %v)\n", data.CurrentTemp, data.Sensor)
	}

	if data.PowerOn {
		fmt.Print("# Thermostat control is ")
		if !data.ThermostatOn {
			fmt.Println(data.OFF)
		} else {
			fmt.Println(data.ON)
		}
		fmt.Printf("# Target temperature is "+tempFormatter+"\n", data.TargetTemp)
		fmt.Printf("# Hystheresis is "+tempFormatter+"\n", data.Hysteresis)
		fmt.Print("# Heat should be ")
		if data.HeatOn {
			fmt.Print(data.ON)
		} else {
			fmt.Print(data.OFF)
		}
		fmt.Print(" (and is ")
		if data.HeatReading {
			fmt.Println(data.ON, ")")
		} else {
			fmt.Println(data.OFF, ")")
		}
	}
}

func readConfig() {
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

		data.PowerOn = powerOnSaved
		data.ThermostatOn = thermostatOnSaved
		data.HeatOn = heatOnSaved
		data.Sensor = sensorSaved
		data.TargetTemp = targetTempSaved
		data.Hysteresis = hysteresisSaved

	} else {
		fmt.Println("Config file does not exist")
	}
}

func writeConfig() {
	configFile, err := os.Create(configFileName)
	if err == nil {
		fmt.Fprintf(configFile, "%v,%v,%v,%v,%v,%v\n", data.PowerOn, data.ThermostatOn, data.HeatOn, data.Sensor, data.TargetTemp, data.Hysteresis)
	}
	configFile.Close()
	log.Println("Config updated:")
	log.Println("  - powerOn is", data.PowerOn)
	log.Println("  - heatOn is", data.HeatOn)
	log.Println("  - thermostatOn", data.ThermostatOn)
	log.Println("  - sensor is", data.Sensor)
	log.Println("  - targetTemp is", data.TargetTemp)
	log.Println("  - hysteresis is", data.Hysteresis)
}

func main() {

	args := os.Args
	if len(args) >= 2 {
		server.WEBPORT = args[1]
	}
	server.HEADER_PAGE_TITLE = "Caldera control and report page"
	log.Printf("Initializing %s with web port='%v'", args[0], server.WEBPORT)
	server.StartWeb()

	executable, err := os.Executable()
	check(err)
	data.LogfileName = executable + ".log"

	logfile, err := os.OpenFile(data.LogfileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}

	log.SetOutput(logfile)
	log.Println(">>> Starting system")

	log.Println("Starting thermostat and all")
	defer logfile.Close()

	log.Println("Configuring rpio ...")
	check(rpio.Open())
	data.PowerPin1.Output()
	data.PowerPin2.Output()
	defer data.PowerPin1.Input()
	defer data.PowerPin2.Input()
	data.HeatPin.Output()
	defer data.HeatPin.Input()
	data.ReadPowerPin.Input()
	data.ReadPowerPin.PullUp()
	data.ReadHeatPin.Input()
	data.ReadHeatPin.PullUp()
	log.Println("Done configuring rpio ...")

	readConfig()

	if data.PowerOn {
		data.SetPower(data.ON)
	} else {
		data.SetPower(data.OFF)
	}
	if data.HeatOn {
		data.SetHeat(data.ON)
	} else {
		data.SetHeat(data.OFF)
	}

	writeConfig()

	// Thermostat loop
	go func() {
		nextRetry := sensorRetry
		for {
			data.ReadPower()
			data.ReadHeat()
			data.ReadTemp()
			if data.ErrorInTemp {
				// Oops, there has been an error measuring the temperature
				if data.ThermostatOn {
					data.SetHeat(data.OFF)
				}
				time.Sleep(nextRetry)
				nextRetry = (nextRetry * 3) / 2 // So we increase the wait time progressively in cummulative errors
				if nextRetry > maxSensorRetry {
					nextRetry = maxSensorRetry
				}
				continue
			} else {
				nextRetry = sensorRetry
			}
			log.Println("Current temp is ", data.CurrentTemp)
			if data.PowerReading && data.ThermostatOn {
				if data.CurrentTemp <= data.TargetTemp-data.Hysteresis && !data.HeatOn {
					data.SetHeat(data.ON)
				} else if data.CurrentTemp >= data.TargetTemp+data.Hysteresis && data.HeatOn {
					data.SetHeat(data.OFF)
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
				oldTemp := data.TargetTemp
				newTemp, err := strconv.ParseFloat(command[1], 64)
				if err != nil {
					fmt.Println("Wrong target temperature: ", command[1])
					continue
				}
				data.TargetTemp = newTemp
				str = fmt.Sprintf("Target temperature changed, old temparture was %.2f, new temperature is %.2f", oldTemp, data.TargetTemp)
			case "changeHyst":
				if len(command) != 2 {
					fmt.Println("Missing hysteresis, syntax is: changeTemp <hyst>")
					continue
				}
				oldHyst := data.Hysteresis
				newHyst, err := strconv.ParseFloat(command[1], 64)
				if err != nil {
					fmt.Println("Wrong hystheresis: ", command[1])
					continue
				}
				data.Hysteresis = newHyst
				str = fmt.Sprintf("Hystheresis changed, old hysteresis was %.2f, new hysteresis is %.2f", oldHyst, newHyst)
			case "changeSensor":
				if len(command) != 2 {
					fmt.Println("Missing new sensor, syntax is: changeSensor <sensor>")
					continue
				}
				oldSensor := data.Sensor
				data.Sensor = command[1]
				str = "Sensor changed, old sensor was " + oldSensor + ", new sensor is " + command[1]
				data.ReadTemp()
			case "pauseThermostat":
				data.ThermostatOn = false
				data.SetHeat(data.OFF)
				str = "Thermostat function now paused (and heat stopped)"
			case "resumeThermostat":
				data.ThermostatOn = true
				fmt.Printf("Thermostat function resumed\n")
				str = "Thermostat function now resumed"
			case "heaterOff":
				data.SetHeat(data.OFF)
				str = "Heat manually disconnected"
			case "heaterOn":
				data.SetHeat(data.ON)
				str = "Heat manually connected"
			case "powerOff":
				data.SetPower(data.OFF)
				str = "Power manually disconnected"
			case "powerOn":
				data.SetPower(data.ON)
				str = "Power manually connected"
			case "help":
				fmt.Println("COMMANDS:")
				fmt.Println("status - prints current status")
				fmt.Println("changeTemp <temp> - sets a new target temperature, e.g. 21.5")
				fmt.Println("changeHyst <hyst> - sets a new hysteresis, e.g. 0.1")
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
			writeConfig()
		}
	}

}
