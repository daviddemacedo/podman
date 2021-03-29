package generate

import (
	"strconv"
	"strings"

	"github.com/containers/podman/v3/pkg/systemd/define"
	"github.com/pkg/errors"
)

// minTimeoutStopSec is the minimal stop timeout for generated systemd units.
// Once exceeded, processes of the services are killed and the cgroup(s) are
// cleaned up.
const minTimeoutStopSec = 60

// validateRestartPolicy checks that the user-provided policy is valid.
func validateRestartPolicy(restart string) error {
	for _, i := range define.RestartPolicies {
		if i == restart {
			return nil
		}
	}
	return errors.Errorf("%s is not a valid restart policy", restart)
}

const headerTemplate = `# {{{{.ServiceName}}}}.service
{{{{- if (eq .GenerateNoHeader false) }}}}
# autogenerated by Podman {{{{.PodmanVersion}}}}
{{{{- if .TimeStamp}}}}
# {{{{.TimeStamp}}}}
{{{{- end}}}}
{{{{- end}}}}

[Unit]
Description=Podman {{{{.ServiceName}}}}.service
Documentation=man:podman-generate-systemd(1)
Wants=network.target
After=network-online.target
RequiresMountsFor={{{{.GraphRoot}}}} {{{{.RunRoot}}}}
`

// filterPodFlags removes --pod, --pod-id-file and --infra-conmon-pidfile from the specified command.
// argCount is the number of last arguments which should not be filtered, e.g. the container entrypoint.
func filterPodFlags(command []string, argCount int) []string {
	processed := []string{}
	for i := 0; i < len(command)-argCount; i++ {
		s := command[i]
		if s == "--pod" || s == "--pod-id-file" || s == "--infra-conmon-pidfile" {
			i++
			continue
		}
		if strings.HasPrefix(s, "--pod=") ||
			strings.HasPrefix(s, "--pod-id-file=") ||
			strings.HasPrefix(s, "--infra-conmon-pidfile=") {
			continue
		}
		processed = append(processed, s)
	}
	processed = append(processed, command[len(command)-argCount:]...)
	return processed
}

// filterCommonContainerFlags removes --conmon-pidfile, --cidfile and --cgroups from the specified command.
// argCount is the number of last arguments which should not be filtered, e.g. the container entrypoint.
func filterCommonContainerFlags(command []string, argCount int) []string {
	processed := []string{}
	for i := 0; i < len(command)-argCount; i++ {
		s := command[i]

		switch {
		case s == "--conmon-pidfile", s == "--cidfile", s == "--cgroups":
			i++
			continue
		case strings.HasPrefix(s, "--conmon-pidfile="),
			strings.HasPrefix(s, "--cidfile="),
			strings.HasPrefix(s, "--cgroups="):
			continue
		}
		processed = append(processed, s)
	}
	processed = append(processed, command[len(command)-argCount:]...)
	return processed
}

// escapeSystemdArguments makes sure that all arguments with at least one whitespace
// are quoted to make sure those are interpreted as one argument instead of
// multiple ones. Also make sure to escape all characters which have a special
// meaning to systemd -> $,% and \
// see: https://www.freedesktop.org/software/systemd/man/systemd.service.html#Command%20lines
func escapeSystemdArguments(command []string) []string {
	for i := range command {
		command[i] = strings.ReplaceAll(command[i], "$", "$$")
		command[i] = strings.ReplaceAll(command[i], "%", "%%")
		if strings.ContainsAny(command[i], " \t") {
			command[i] = strconv.Quote(command[i])
		} else if strings.Contains(command[i], `\`) {
			// strconv.Quote also escapes backslashes so
			// we should replace only if strconv.Quote was not used
			command[i] = strings.ReplaceAll(command[i], `\`, `\\`)
		}
	}
	return command
}

func removeDetachArg(args []string, argCount int) []string {
	// "--detach=false" could also be in the container entrypoint
	// split them off so we do not remove it there
	realArgs := args[len(args)-argCount:]
	flagArgs := removeArg("-d=false", args[:len(args)-argCount])
	flagArgs = removeArg("--detach=false", flagArgs)
	return append(flagArgs, realArgs...)
}

func removeReplaceArg(args []string, argCount int) []string {
	// "--replace=false" could also be in the container entrypoint
	// split them off so we do not remove it there
	realArgs := args[len(args)-argCount:]
	flagArgs := removeArg("--replace=false", args[:len(args)-argCount])
	return append(flagArgs, realArgs...)
}

func removeArg(arg string, args []string) []string {
	newArgs := []string{}
	for _, a := range args {
		if a != arg {
			newArgs = append(newArgs, a)
		}
	}
	return newArgs
}
