package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
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
	opts := client.EventsListOptions{
		Since: "1h",
		Filters: make(client.Filters).Add("event",
			// Containers
			"health_status",
			"kill",
			"restart",
			"start",
			"stop",
			"update",
			// Images
			"delete",
			"import",
			"load",
			"prune",
			// Plugins
			"install",
			"remove",
			// Volumes
			"create",
			"destroy",
		),
	}
	result := cli.Events(ctx, opts)
	exporterStatus.Docker = "Waiting for events"

	return result.Messages, result.Err
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
