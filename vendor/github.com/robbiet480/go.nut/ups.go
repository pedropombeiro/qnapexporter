package nut

import (
	"fmt"
	"strconv"
	"strings"
	"regexp"
)

// UPS contains information about a specific UPS provided by the NUT instance.
type UPS struct {
	Name           string
	Description    string
	Master         bool
	NumberOfLogins int
	Clients        []string
	Variables      []Variable
	Commands       []Command
	nutClient      *Client
}

// Variable describes a single variable related to a UPS.
type Variable struct {
	Name          string
	Value         interface{}
	Type          string
	Description   string
	Writeable     bool
	MaximumLength int
	OriginalType  string
}

// Command describes an available command for a UPS.
type Command struct {
	Name        string
	Description string
}

// NewUPS takes a UPS name and NUT client and returns an instantiated UPS struct.
func NewUPS(name string, client *Client) (UPS, error) {
	newUPS := UPS{
		Name:      name,
		nutClient: client,
	}
	_, err := newUPS.GetClients()
	if err != nil {
		return newUPS, err
	}
	_, err = newUPS.GetCommands()
	if err != nil {
		return newUPS, err
	}
	_, err = newUPS.GetDescription()
	if err != nil {
		return newUPS, err
	}
	_, err = newUPS.GetNumberOfLogins()
	if err != nil {
		return newUPS, err
	}
	_, err = newUPS.GetVariables()
	if err != nil {
		return newUPS, err
	}
	return newUPS, err
}

// GetNumberOfLogins returns the number of clients which have done LOGIN for this UPS.
func (u *UPS) GetNumberOfLogins() (int, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("GET NUMLOGINS %s", u.Name))
	if err != nil {
		return 0, err
	}
	atoi, err := strconv.Atoi(strings.TrimPrefix(resp[0], fmt.Sprintf("NUMLOGINS %s ", u.Name)))
	if err != nil {
		return 0, err
	}
	u.NumberOfLogins = atoi
	return atoi, nil
}

// GetClients returns a list of NUT clients.
func (u *UPS) GetClients() ([]string, error) {
	clientsList := []string{}
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("LIST CLIENT %s", u.Name))
	if err != nil {
		return clientsList, err
	}
	linePrefix := fmt.Sprintf("CLIENT %s ", u.Name)
	for _, line := range resp[1 : len(resp)-1] {
		clientsList = append(clientsList, strings.TrimPrefix(line, linePrefix))
	}
	u.Clients = clientsList
	return clientsList, nil
}

// CheckIfMaster returns true if the session is authenticated with the master permission set.
func (u *UPS) CheckIfMaster() (bool, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("MASTER %s", u.Name))
	if err != nil {
		return false, err
	}
	if resp[0] == "OK" {
		u.Master = true
		return true, nil
	}
	return false, nil
}

// GetDescription the value of "desc=" from ups.conf for this UPS. If it is not set, upsd will return "Unavailable".
func (u *UPS) GetDescription() (string, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("GET UPSDESC %s", u.Name))
	if err != nil {
		return "", err
	}
	description := strings.TrimPrefix(strings.Replace(resp[0], `"`, "", -1), fmt.Sprintf(`UPSDESC %s `, u.Name))
	u.Description = description
	return description, nil
}

// GetVariables returns a slice of Variable structs for the UPS.
func (u *UPS) GetVariables() ([]Variable, error) {
	vars := []Variable{}
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("LIST VAR %s", u.Name))
	if err != nil {
		return vars, err
	}
	offset := fmt.Sprintf("VAR %s ", u.Name)
	for _, line := range resp[1 : len(resp)-1] {
		newVar := Variable{}
		cleanedLine := strings.TrimPrefix(line, offset)
		splitLine := strings.Split(cleanedLine, `"`)
		splitLine[1] = strings.Trim(splitLine[1], " ")
		newVar.Name = strings.TrimSuffix(splitLine[0], " ")
		newVar.Value = splitLine[1]

		description, err := u.GetVariableDescription(newVar.Name)
		if err != nil {
			return vars, err
		}
		newVar.Description = description
		varType, writeable, maximumLength, err := u.GetVariableType(newVar.Name)
		if err != nil {
			return vars, err
		}
		newVar.Type = varType
		newVar.Writeable = writeable
		newVar.MaximumLength = maximumLength

		if splitLine[1] == "enabled" {
			newVar.Value = true
			newVar.Type = "BOOLEAN"
		}
		if splitLine[1] == "disabled" {
			newVar.Value = false
			newVar.Type = "BOOLEAN"
		}

		matched, _ := regexp.MatchString(`^-?[0-9\.]+$`, splitLine[1])
		if matched {
			if strings.Count(splitLine[1], ".") == 1 {
				converted, err := strconv.ParseFloat(splitLine[1], 64)
				if err == nil {
					newVar.Value = converted
					newVar.Type = "FLOAT_64"
					newVar.OriginalType = varType
				}
			} else {
				converted, err := strconv.ParseInt(splitLine[1], 10, 64)
				if err == nil {
					newVar.Value = converted
					newVar.Type = "INTEGER"
					newVar.OriginalType = varType
				}
			}
		}

		/* Failed conversion or not a numeric value - coax to STRING */
		if err != nil || !matched {
			newVar.Type = "STRING"
			newVar.OriginalType = varType
		}

		vars = append(vars, newVar)
	}
	u.Variables = vars
	return vars, nil
}

// GetVariableDescription returns a string that gives a brief explanation for the given variableName.
// upsd may return "Unavailable" if the file which provides this description is not installed.
func (u *UPS) GetVariableDescription(variableName string) (string, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("GET DESC %s %s", u.Name, variableName))
	if err != nil {
		return "", err
	}
	trimmedLine := strings.TrimPrefix(resp[0], fmt.Sprintf("DESC %s %s ", u.Name, variableName))
	description := strings.Replace(trimmedLine, `"`, "", -1)
	return description, nil
}

// GetVariableType returns the variable type, writeability and maximum length for the given variableName.
func (u *UPS) GetVariableType(variableName string) (string, bool, int, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("GET TYPE %s %s", u.Name, variableName))
	if err != nil {
		return "UNKNOWN", false, -1, err
	}
	trimmedLine := strings.TrimPrefix(resp[0], fmt.Sprintf("TYPE %s %s ", u.Name, variableName))
	splitLine := strings.Split(trimmedLine, " ")
	writeable := (splitLine[0] == "RW")
	varType := "UNKNOWN"
	maximumLength := 0
	if writeable {
		varType = splitLine[1]
		if strings.HasPrefix(varType, "STRING:") {
			splitType := strings.Split(varType, ":")
			varType = splitType[0]
			maximumLength, err = strconv.Atoi(splitType[1])
			if err != nil {
				return varType, writeable, -1, err
			}
		}
	} else {
		varType = splitLine[0]
	}
	return varType, writeable, maximumLength, nil
}

// GetCommands returns a slice of Command structs for the UPS.
func (u *UPS) GetCommands() ([]Command, error) {
	commandsList := []Command{}
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("LIST CMD %s", u.Name))
	if err != nil {
		return commandsList, err
	}
	linePrefix := fmt.Sprintf("CMD %s ", u.Name)
	for _, line := range resp[1 : len(resp)-1] {
		cmdName := strings.TrimPrefix(line, linePrefix)
		cmd := Command{
			Name: cmdName,
		}
		description, err := u.GetCommandDescription(cmdName)
		if err != nil {
			return commandsList, err
		}
		cmd.Description = description
		commandsList = append(commandsList, cmd)
	}
	u.Commands = commandsList
	return commandsList, nil
}

// GetCommandDescription returns a string that gives a brief explanation for the given commandName.
func (u *UPS) GetCommandDescription(commandName string) (string, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("GET CMDDESC %s %s", u.Name, commandName))
	if err != nil {
		return "", err
	}
	trimmedLine := strings.TrimPrefix(resp[0], fmt.Sprintf("CMDDESC %s %s ", u.Name, commandName))
	description := strings.Replace(trimmedLine, `"`, "", -1)
	return description, err
}

// SetVariable sets the given variableName to the given value on the UPS.
func (u *UPS) SetVariable(variableName, value string) (bool, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf(`SET VAR %s %s "%s"`, u.Name, variableName, value))
	if err != nil {
		return false, err
	}
	if resp[0] == "OK" {
		return true, nil
	}
	return false, nil
}

// SendCommand sends a command to the UPS.
func (u *UPS) SendCommand(commandName string) (bool, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("INSTCMD %s %s", u.Name, commandName))
	if err != nil {
		return false, err
	}
	if resp[0] == "OK" {
		return true, nil
	}
	return false, nil
}

// ForceShutdown sets the FSD flag on the UPS.
//
// This requires "upsmon master" in upsd.users, or "FSD" action granted in upsd.users
//
// upsmon in master mode is the primary user of this function. It sets this "forced shutdown" flag on any UPS when it plans to power it off. This is done so that slave systems will know about it and shut down before the power disappears.
//
// Setting this flag makes "FSD" appear in a STATUS request for this UPS. Finding "FSD" in a status request should be treated just like a "OB LB".
//
// It should be noted that FSD is currently a latch - once set, there is no way to clear it short of restarting upsd or dropping then re-adding it in the ups.conf. This may cause issues when upsd is running on a system that is not shut down due to the UPS event.
func (u *UPS) ForceShutdown() (bool, error) {
	resp, err := u.nutClient.SendCommand(fmt.Sprintf("FSD %s", u.Name))
	if err != nil {
		return false, err
	}
	if resp[0] == "OK FSD-SET" {
		return true, nil
	}
	return false, nil
}
