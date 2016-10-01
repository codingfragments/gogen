package run

import (
	"github.com/coccyx/gogen/generator"
	config "github.com/coccyx/gogen/internal"
	"github.com/coccyx/gogen/outputter"
	"github.com/coccyx/gogen/timer"
)

// Run runs the mainline of the program
func Run(c *config.Config) {
	c.Log.Info("Starting ReadOutThread")
	go outputter.ROT(c)
	c.Log.Info("Starting Timers")
	timerdone := make(chan int)
	gq := make(chan *config.GenQueueItem)
	gqs := make(chan int)
	oq := make(chan *config.OutQueueItem)
	oqs := make(chan int)
	gens := 0
	outs := 0
	timers := 0
	for i := 0; i < len(c.Samples); i++ {
		s := c.Samples[i]
		if !s.Disabled {
			t := timer.Timer{S: s, GQ: gq, OQ: oq, Done: timerdone}
			go t.NewTimer()
			timers++
		}
	}
	c.Log.Infof("%d Timers started", timers)

	c.Log.Infof("Starting Generators")
	for i := 0; i < c.Global.GeneratorWorkers; i++ {
		c.Log.Infof("Starting Generator %d", i)
		go generator.Start(gq, gqs)
		gens++
	}

	c.Log.Infof("Starting Outputters")
	for i := 0; i < c.Global.OutputWorkers; i++ {
		c.Log.Infof("Starting Outputter %d", i)
		go outputter.Start(oq, oqs)
		outs++
	}

	// time.Sleep(1000 * time.Millisecond)

	// Check if any timers are done
Loop1:
	for {
		select {
		case <-timerdone:
			timers--
			c.Log.Debugf("Timer done, timers left %d", timers)
			if timers == 0 {
				break Loop1
			}
		}
	}

	// Close our channels to signal to the workers to shut down when the queue is clear
	c.Log.Infof("Timers all done, closing generating queue")
	close(gq)

	// Check for all the workers to signal back they're done
Loop2:
	for {
		select {
		case <-gqs:
			gens--
			c.Log.Debugf("Gen done, gens left %d", gens)
			if gens == 0 {
				break Loop2
			}
		}
	}

	// Close our output channel to signal to outputters we're done
	close(oq)
Loop3:
	for {
		select {
		case <-oqs:
			outs--
			c.Log.Debugf("Out done, outs left %d", outs)
			if outs == 0 {
				break Loop3
			}
		}
	}

	// time.Sleep(100 * time.Millisecond)
}