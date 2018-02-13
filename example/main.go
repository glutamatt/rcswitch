package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/glutamatt/rcswitch"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

var pin = flag.String("pin", "27", "Number of the pin the transmitter or receiver is attached to")
var hook = flag.String("hook", "http://localhost:8008/signal?type=", "Url to post events")

func main() {
	flag.Parse()
	_, err := host.Init()
	if err != nil {
		log.Fatal(err)
	}
	p := gpioreg.ByName(*pin)
	if err := p.In(gpio.PullDown, gpio.BothEdges); err != nil {
		log.Fatal(err)
	}

	sw := rcswitch.New(p)

	codes := make(chan int)
	porteLimiter := NewLimiter(3 * time.Second)
	btnLimiter := NewLimiter(3 * time.Second)

	go func() {
		for code := range codes {
			switch code {
			case 6729992:
				btnLimiter.Do(func() {
					callHook("button")
				})

			case 2448968:
				porteLimiter.Do(func() {
					callHook("door")
				})
			}
		}
	}()

	sw.Scan(codes)
}

func callHook(event string) {
	if _, err := http.Get(*hook + event); err != nil {
		log.Println(err.Error())
	}
}

type limiter struct {
	limit time.Duration
	block chan bool
}

func NewLimiter(t time.Duration) limiter {
	l := limiter{
		limit: t,
		block: make(chan bool, 1),
	}

	l.block <- true
	return l
}

func (l *limiter) Do(f func()) {
	select {
	case <-l.block:
		go func() {
			time.Sleep(l.limit)
			l.block <- true
		}()
		f()
	default:
		//pass
	}
}
