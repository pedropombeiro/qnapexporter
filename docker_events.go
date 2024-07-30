package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/pedropombeiro/qnapexporter/lib/exporter"
	"github.com/pedropombeiro/qnapexporter/lib/notifications"
)

func handleDockerEvents(ctx context.Context, args httpServerArgs, annotator notifications.Annotator, exporterStatus *exporter.Status) error {
	exporterStatus.Docker = "Connecting..."

	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		exporterStatus.Docker = err.Error()
		args.logger.Println(err)
		return err
	}

	msgs, errs := dockerEvents(ctx, cli, exporterStatus)

	for {
		select {
		case err := <-errs:
			if err != nil {
				exporterStatus.Docker = err.Error()
				args.logger.Println(err)

				select {
				case <-time.After(10 * time.Second):
					// Wait for 10 seconds before retrying connection to Docker daemon
					msgs, errs = dockerEvents(ctx, cli, exporterStatus)
					break
				case <-ctx.Done():
					break
				}
			}
		case msg := <-msgs:
			t := time.Unix(0, msg.TimeNano)
			m := strings.Join([]string{string(msg.Type), string(msg.Action), msg.Actor.ID, formatDockerActorAttributes(msg.Actor.Attributes)}, " ")
			exporterStatus.Docker = m
			args.logger.Printf("%v: %s\n", t, m)
			_, _ = annotator.Post(m, t)
		case <-ctx.Done():
			exporterStatus.Docker = "Done"
			return nil
		}
	}
}

func dockerEvents(ctx context.Context, cli *client.Client, exporterStatus *exporter.Status) (<-chan events.Message, <-chan error) {
	opts := events.ListOptions{
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

	return msgs, errs
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
