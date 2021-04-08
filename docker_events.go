package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"gitlab.com/pedropombeiro/qnapexporter/lib/exporter"
	"gitlab.com/pedropombeiro/qnapexporter/lib/notifications"
)

func handleDockerEvents(args httpServerArgs, annotator notifications.Annotator, exporterStatus *exporter.Status) error {
	exporterStatus.Docker = "Connecting..."

	// Setup our Ctrl+C handler
	exitCh := make(chan os.Signal, 1)
	signal.Notify(exitCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		exporterStatus.Docker = err.Error()
		args.logger.Println(err)
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-exitCh
		log.Println("Program aborted, stopping Docker listener...")
		cancel()
	}()

	opts := types.EventsOptions{
		Since: "1h",
		Filters: filters.NewArgs(
			// Containers
			filters.Arg("event", "health_status"),
			filters.Arg("event", "kill"),
			filters.Arg("event", "restart"),
			filters.Arg("event", "start"),
			filters.Arg("event", "stop"),
			filters.Arg("event", "update"),
			// Images
			filters.Arg("event", "delete"),
			filters.Arg("event", "import"),
			filters.Arg("event", "load"),
			filters.Arg("event", "prune"),
			// Plugins
			filters.Arg("event", "install"),
			filters.Arg("event", "remove"),
			// Volumes
			filters.Arg("event", "create"),
			filters.Arg("event", "destroy"),
		),
	}
	msgs, errs := cli.Events(ctx, opts)
	exporterStatus.Docker = "Waiting for events"

	for {
		select {
		case err := <-errs:
			if err != nil {
				exporterStatus.Docker = err.Error()
				args.logger.Println(err)
				time.Sleep(10 * time.Second)
			}
		case msg := <-msgs:
			t := time.Unix(0, msg.TimeNano)
			m := strings.Join([]string{msg.Type, msg.Action, msg.Actor.ID, formatDockerActorAttributes(msg.Actor.Attributes)}, " ")
			exporterStatus.Docker = m
			args.logger.Printf("%v: %s\n", t, m)
			_, _ = annotator.Post(m, t)
		case <-ctx.Done():
			exporterStatus.Docker = "Done"
			return nil
		}
	}
}

func formatDockerActorAttributes(attr map[string]string) string {
	var s string
	for k, v := range attr {
		if strings.HasPrefix(k, "com.docker.") {
			continue
		}

		if s != "" {
			s += ","
		}
		s += fmt.Sprintf("%s=%s", k, v)
	}
	return "(" + s + ")"
}
