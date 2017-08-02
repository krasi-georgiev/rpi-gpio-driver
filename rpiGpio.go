package rpiGpio

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gpioPins = []int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 22, 23, 24, 25, 26, 27}
)

const sysfs string = "/sys/class/gpio/"
const sysfsGPIOenable string = sysfs + "export"
const sysfsGPIOdisable string = sysfs + "unexport"

const DefaultDelay = 2
const DefaultPin = "18"
const DefaultType = "timer"

// Controller interface describes the main funcitons when triggering a pin
type Controller interface {
	StartTimer(ch chan string) error
	Toggle(ch chan string) error
}

//NewControl the constructor with some defaults
func NewControl(opts ...func(*Control) error) (*Control, error) {
	ctrl := &Control{}
	for _, o := range opts {
		if err := o(ctrl); err != nil {
			return nil, err
		}
	}

	return ctrl, nil
}

// Control holds all configuration
type Control struct {
	ctype string
	pin   string
	delay time.Duration
}

// SetType is the controller ctype setter
func SetType(d string) func(*Control) error {
	return func(c *Control) error {
		switch strings.TrimSpace(d) {
		case "":
			c.ctype = DefaultType
		case "timer", "toggle":
			c.ctype = strings.TrimSpace(d)
		default:
			return errors.New("Invalid control type:" + d)
		}
		return nil
	}
}

//SetPin the pin on gpio that willbe controlled
func SetPin(d string) func(*Control) error {
	return func(c *Control) error {
		if d == "" {
			c.pin = DefaultPin
			return nil
		}
		for _, v := range gpioPins {
			if strconv.Itoa(v) == d {
				c.pin = d
				return nil
			}
		}
		sort.Ints(gpioPins)
		return fmt.Errorf("Invalid GPIO pin number:%v, choose one of :%v", d, gpioPins)
	}
}

// SetDelay delay between enable and disable timer
func SetDelay(d string) func(*Control) error {
	return func(c *Control) error {
		if d == "" {
			c.delay = DefaultDelay
			return nil
		}
		if t, err := time.ParseDuration(d); err == nil {
			c.delay = t
			return nil
		}
		return fmt.Errorf("Invalid time delay format :%v (use 1ms, 1s, 1m, 1h)", d)
	}
}

func (c *Control) enablePin() error {
	// enable if not already enabled
	if _, err := os.Stat(sysfs + "gpio" + c.pin); os.IsNotExist(err) {
		if _, err := os.Stat(sysfsGPIOenable); os.IsNotExist(err) {
			return err
		}
		if err := ioutil.WriteFile(sysfsGPIOenable, []byte(c.pin), 0644); err != nil {
			return err
		}
		if err := ioutil.WriteFile(sysfs+"gpio"+c.pin+"/direction", []byte("out"), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (c *Control) disablePin() {
	if _, err := os.Stat(sysfs + "gpio" + c.pin); os.IsNotExist(err) {
		// it is already disabled so nothing else to do, bail out
		return
	}

	err := ioutil.WriteFile(sysfsGPIOdisable, []byte(c.pin), 0644)
	if err != nil {
		log.Printf("Oops can't disable pin %v because %v", c.pin, err)
	}
}

// Run executes the control with the initiated settings
func (c *Control) Run() error {
	switch c.ctype {
	case "timer":
		return c.startTimer()
	case "toggle":
		return c.toggle()
	default:
		return fmt.Errorf("Invalid control type:%v", c.ctype)
	}
}

func (c *Control) startTimer() error {
	if err := c.enablePin(); err != nil {
		log.Printf("I couldn't enable pin %v, because %v", c.pin, err)
		return err
	}
	if err := ioutil.WriteFile(sysfs+"gpio"+c.pin+"/value", []byte("1"), 0644); err != nil {
		return err
	}
	go func() {
		time.Sleep(c.delay)
		if err := ioutil.WriteFile(sysfs+"gpio"+c.pin+"/value", []byte("0"), 0644); err != nil {
			log.Printf("Couldn't disable pin:%v error:%v", c.pin, err)
		}
	}()
	return nil
}

func (c *Control) toggle() error {
	if err := c.enablePin(); err != nil {
		log.Printf("I couldn't enable pin %v, because %v", c.pin, err)
	}

	d, err := ioutil.ReadFile(sysfs + "gpio" + c.pin + "/value")
	if err != nil {
		log.Printf("Oh boy can't read the status of pin	%v becasue I don't have my glasses and %v", c.pin, err)
	}

	if string(d) == "1\n" {
		if err := ioutil.WriteFile(sysfs+"gpio"+c.pin+"/value", []byte("0"), 0644); err != nil {
			return err
		}
		return nil
	}
	if err := ioutil.WriteFile(sysfs+"gpio"+c.pin+"/value", []byte("1"), 0644); err != nil {
		return err
	}
	return nil
}
