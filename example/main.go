package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/marcelvanwaaijen/workerpool"
)

func main() {
	// create a context with a timeout of 5 seconds
	ctx := context.Background()
	tctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	// when pressing CTRL+C before timeout has expired, call the cancel function
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			cancel()
		}
	}()

	// create a new workerpool with 2 workers and a queue of 10 items
	wp := workerpool.New(tctx, 2, 10, doWork)

	// start with looking for work
	wp.Start()

	// add 20 items to the work queue
	for i := 1; i < 20; i++ {
		if err := wp.Add(i); err != nil {
			log.Print(err)
			break
		}
	}

	// wait until the queue has been cleared
	if err := wp.Wait(); err != nil {
		log.Print(err)
	}
}

func doWork(_ context.Context, work int) error {
	log.Printf("starting work on item %d", work)
	time.Sleep(3 * time.Second)
	log.Printf("done working on item %d", work)
	return nil
}
