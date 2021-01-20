// Package nut provides a Golang interface for interacting with Network UPS Tools (NUT).
//
// It communicates with NUT over the TCP protocol
package nut

import (
	"bufio"
	"fmt"
	"net"
	"strings"
)

// Client contains information about the NUT server as well as the connection.
type Client struct {
	Version         string
	ProtocolVersion string
	Hostname        net.Addr
	conn            *net.TCPConn
}

// Connect accepts a hostname/IP string and creates a connection to NUT, returning a Client.
func Connect(hostname string) (Client, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:3493", hostname))
	if err != nil {
		return Client{}, err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return Client{}, err
	}
	client := Client{
		Hostname: conn.RemoteAddr(),
		conn:     conn,
	}
	client.GetVersion()
	client.GetNetworkProtocolVersion()
	return client, nil
}

// Disconnect gracefully disconnects from NUT by sending the LOGOUT command.
func (c *Client) Disconnect() (bool, error) {
	logoutResp, err := c.SendCommand("LOGOUT")
	if err != nil {
		return false, err
	}
	if logoutResp[0] == "OK Goodbye" || logoutResp[0] == "Goodbye..." {
		return true, nil
	}
	return false, nil
}

// ReadResponse is a convenience function for reading newline delimited responses.
func (c *Client) ReadResponse(endLine string, multiLineResponse bool) (resp []string, err error) {
	connbuff := bufio.NewReader(c.conn)
	response := []string{}

	for {
		line, err := connbuff.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}
		if len(line) > 0 {
			cleanLine := strings.TrimSuffix(line, "\n")
			lines := strings.Split(cleanLine, "\n")
			response = append(response, lines...)
			if line == endLine || multiLineResponse == false {
				break
			}
		}
	}

	return response, err
}

// SendCommand sends the string cmd to the device, and returns the response.
func (c *Client) SendCommand(cmd string) (resp []string, err error) {
	cmd = fmt.Sprintf("%v\n", cmd)
	endLine := fmt.Sprintf("END %s", cmd)
	if strings.HasPrefix(cmd, "USERNAME ") || strings.HasPrefix(cmd, "PASSWORD ") || strings.HasPrefix(cmd, "SET ") || strings.HasPrefix(cmd, "HELP ") || strings.HasPrefix(cmd, "VER ") || strings.HasPrefix(cmd, "NETVER ") {
		endLine = "OK\n"
	}
	_, err = fmt.Fprint(c.conn, cmd)
	if err != nil {
		return []string{}, err
	}

	resp, err = c.ReadResponse(endLine, strings.HasPrefix(cmd, "LIST "))
	if err != nil {
		return []string{}, err
	}

	if strings.HasPrefix(resp[0], "ERR ") {
		return []string{}, errorForMessage(strings.Split(resp[0], " ")[1])
	}

	return resp, nil
}

// Authenticate accepts a username and passwords and uses them to authenticate the existing NUT session.
func (c *Client) Authenticate(username, password string) (bool, error) {
	usernameResp, err := c.SendCommand(fmt.Sprintf("USERNAME %s", username))
	if err != nil {
		return false, err
	}
	passwordResp, err := c.SendCommand(fmt.Sprintf("PASSWORD %s", password))
	if err != nil {
		return false, err
	}
	if usernameResp[0] == "OK" && passwordResp[0] == "OK" {
		return true, nil
	}
	return false, nil
}

// GetUPSList returns a list of all UPSes provided by this NUT instance.
func (c *Client) GetUPSList() ([]UPS, error) {
	upsList := []UPS{}
	resp, err := c.SendCommand("LIST UPS")
	if err != nil {
		return upsList, err
	}
	for _, line := range resp {
		if strings.HasPrefix(line, "UPS ") {
			splitLine := strings.Split(strings.TrimPrefix(line, "UPS "), `"`)
			newUPS, err := NewUPS(strings.TrimSuffix(splitLine[0], " "), c)
			if err != nil {
				return upsList, err
			}
			upsList = append(upsList, newUPS)
		}
	}
	return upsList, err
}

// Help returns a list of the commands supported by NUT.
func (c *Client) Help() (string, error) {
	helpResp, err := c.SendCommand("HELP")
	return helpResp[0], err
}

// GetVersion returns the the version of the server currently in use.
func (c *Client) GetVersion() (string, error) {
	versionResponse, err := c.SendCommand("VER")
	c.Version = versionResponse[0]
	return versionResponse[0], err
}

// GetNetworkProtocolVersion returns the version of the network protocol currently in use.
func (c *Client) GetNetworkProtocolVersion() (string, error) {
	versionResponse, err := c.SendCommand("NETVER")
	c.ProtocolVersion = versionResponse[0]
	return versionResponse[0], err
}
