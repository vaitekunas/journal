package server

import (
	"bytes"
	"fmt"
	"github.com/fatih/color"
	"github.com/vaitekunas/lentele"
	"github.com/vaitekunas/unixsock"
	"reflect"
	"sort"
	"strings"
	"time"
)

// ManagementConsole handles commands received over the unix socket
type ManagementConsole interface {

	// CmdStatistics displays various statistics
	CmdStatistics(unixsock.Args) *unixsock.Response

	// CmdLogsList list all available logfiles and their archives
	CmdLogsList(unixsock.Args) *unixsock.Response

	// CmdRemoteAdd adds a remote backend
	CmdRemoteAdd(unixsock.Args) *unixsock.Response

	// CmdRemoteList lists all active remote backends
	CmdRemoteList(unixsock.Args) *unixsock.Response

	// CmdRemoteRemove removes a remote backend
	CmdRemoteRemove(unixsock.Args) *unixsock.Response

	// CmdTokensAdd adds a new token for a service/instance
	CmdTokensAdd(unixsock.Args) *unixsock.Response

	// CmdTokensListInstances lists all permitted instances of a service
	CmdTokensListInstances(unixsock.Args) *unixsock.Response

	// CmdTokensListServices lists all permitted services
	CmdTokensListServices(unixsock.Args) *unixsock.Response

	// CmdTokensRemoveInstance removes the token of a service/instance
	CmdTokensRemoveInstance(unixsock.Args) *unixsock.Response

	// CmdTokensRemoveService removes the token of all instances of a service
	CmdTokensRemoveService(unixsock.Args) *unixsock.Response

	// Execute is the executor of management console commands
	Execute(string, unixsock.Args) *unixsock.Response
}

// NewConsole creates a new management console for the log server
func NewConsole(server *LogServer) ManagementConsole {

	return &managementConsole{
		logserver: server,
	}
}

// managementConsole handles commands received over the unix socket
type managementConsole struct {
	banner    string
	logserver *LogServer
}

// Execute is the executor of management console commands
func (m *managementConsole) Execute(cmd string, args unixsock.Args) *unixsock.Response {

	fmt.Printf(" ▶ [%s] Received command [%s]\n", time.Now().Format("2006-01-02 15:04:05"), bold(strings.ToLower(cmd)))

	switch strings.ToLower(cmd) {
	case "statistics":
		return m.CmdStatistics(args)
	case "tokens.add":
		return m.CmdTokensAdd(args)
	case "tokens.remove.instance":
		return m.CmdTokensRemoveInstance(args)
	case "tokens.remove.service":
		return m.CmdTokensRemoveService(args)
	case "tokens.list.instances":
		return m.CmdTokensListInstances(args)
	case "tokens.list.services":
		return m.CmdTokensListServices(args)
	case "logs.list":
		return m.CmdLogsList(args)
	case "remote.add":
		return m.CmdRemoteAdd(args)
	case "remote.remove":
		return m.CmdRemoteRemove(args)
	case "remote.list":
		return m.CmdRemoteList(args)
	default:
		return &unixsock.Response{
			Status: "failure",
			Error:  fmt.Errorf("Execute: unknown command '%s'", cmd).Error(),
		}
	}

}

// arg is a helper struct used to for slices of required arguments
type arg struct {
	Name string
	Kind reflect.Kind
}

// validArguments verifies that all the required arguments have been provided
func validArguments(args unixsock.Args, required []arg) bool {
	for _, f := range required {
		x, ok := args[f.Name]
		if !ok {
			return false
		}

		if reflect.TypeOf(x).Kind() != f.Kind {
			return false
		}
	}
	return true
}

var respMissingArgs = &unixsock.Response{
	Status: "failure",
	Error:  fmt.Errorf("Missing/invalid parameters").Error(),
}

// CmdStatistics displays various log-related statistics
func (m *managementConsole) CmdStatistics(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdTokensAdd adds a new token for a service/instance
func (m *managementConsole) CmdTokensAdd(args unixsock.Args) *unixsock.Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
		arg{"instance", reflect.String},
	}

	// Validate arguments
	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service/instance
	service := args["service"].(string)
	instance := args["instance"].(string)
	token, err := m.logserver.AddToken(service, instance)
	if err != nil {
		return &unixsock.Response{
			Status: "failure",
			Error:  fmt.Errorf("Could not add token: %s", err.Error()).Error(),
		}
	}

	// Prepare table
	table := lentele.New("Service", "Instance", "Token")
	table.AddTitle(fmt.Sprintf("Token created for %s/%s", service, instance))
	table.AddRow("").Insert(service, instance, token).Modify(bold, "Token")
	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	// Successful op
	return &unixsock.Response{
		Status:  "success",
		Payload: buf.String(),
	}

}

// CmdTokensRemoveInstance removes the token of a service/instance
func (m *managementConsole) CmdTokensRemoveInstance(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdTokensRemoveService removes the token of all instances of a service
func (m *managementConsole) CmdTokensRemoveService(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdTokensListInstances lists all permitted instances of a service
func (m *managementConsole) CmdTokensListInstances(args unixsock.Args) *unixsock.Response {

	// Validate arguments
	required := []arg{
		arg{"service", reflect.String},
	}

	if !validArguments(args, required) {
		return respMissingArgs
	}

	// Identify service
	service := strings.ToLower(args["service"].(string))

	// Prepare table
	table := lentele.New("Instance", "Token", "Last IP", "Logs parsed")
	table.AddTitle(fmt.Sprintf("Service %s: permited instances", service))

	for key, token := range m.logserver.tokens {
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			continue
		}
		if parts[0] == service {
			table.AddRow("").Insert(parts[1], fmt.Sprintf("%s...", token[0:10]), "???", "???")
		}
	}

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: buf.String(),
	}
}

// CmdTokensListServices lists all permitted services
func (m *managementConsole) CmdTokensListServices(args unixsock.Args) *unixsock.Response {

	// Prepare statistics
	serviceNames := []string{}
	services := map[string][2]int{}
	for key := range m.logserver.tokens {
		parts := strings.Split(key, "/")
		if len(parts) != 2 {
			continue
		}
		if _, ok := services[parts[0]]; !ok {
			serviceNames = append(serviceNames, parts[0])
			services[parts[0]] = [2]int{}
		}
		counts := services[parts[0]]
		counts[0]++
	}
	sort.Strings(serviceNames)

	busy := func(v interface{}) interface{} {
		return color.New(color.FgRed).Sprint(v)
	}

	// Prepare table
	table := lentele.New("", "Service", "Instances", "Last log entry", "Log entries parsed")
	table.AddTitle("Permitted services")
	for _, name := range serviceNames {
		service := services[name]
		now := time.Now().Format("2006-01-02 15:04")

		table.AddRow("").Insert("●", name, service[0], now, service[1]).Modify(busy, "")

	}

	buf := bytes.NewBuffer([]byte{})
	table.Render(buf, false, true, false, lentele.LoadTemplate("classic"))

	return &unixsock.Response{
		Status:  "success",
		Payload: buf.String(),
	}
}

// CmdLogsList list all available logfiles and their archives
func (m *managementConsole) CmdLogsList(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteAdd adds a remote backend
func (m *managementConsole) CmdRemoteAdd(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteRemove removes a remote backend
func (m *managementConsole) CmdRemoteRemove(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}

// CmdRemoteList lists all active remote backends
func (m *managementConsole) CmdRemoteList(args unixsock.Args) *unixsock.Response {
	return &unixsock.Response{}
}
