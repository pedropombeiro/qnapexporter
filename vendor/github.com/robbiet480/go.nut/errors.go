package nut

import "errors"

// errorForMessage returns an error for the specified NUT error code.
func errorForMessage(message string) (err error) {
	switch message {
	case "ACCESS-DENIED":
		err = errors.New("The client’s host and/or authentication details (username, password) are not sufficient to execute the requested command")
	case "UNKNOWN-UPS":
		err = errors.New("The UPS specified in the request is not known to upsd. This usually means that it didn’t match anything in ups.conf")
	case "VAR-NOT-SUPPORTED":
		err = errors.New("The specified UPS doesn’t support the variable in the request. This is also sent for unrecognized variables which are in a space which is handled by upsd, such as server.*")
	case "CMD-NOT-SUPPORTED":
		err = errors.New("The specified UPS doesn’t support the instant command in the request")
	case "INVALID-ARGUMENT":
		err = errors.New("The client sent an argument to a command which is not recognized or is otherwise invalid in this context. This is typically caused by sending a valid command like GET with an invalid subcommand")
	case "INSTCMD-FAILED":
		err = errors.New("upsd failed to deliver the instant command request to the driver. No further information is available to the client. This typically indicates a dead or broken driver")
	case "SET-FAILED":
		err = errors.New("upsd failed to deliver the set request to the driver. This is just like INSTCMD-FAILED above")
	case "READONLY":
		err = errors.New("The requested variable in a SET command is not writable")
	case "TOO-LONG":
		err = errors.New("The requested value in a SET command is too long")
	case "FEATURE-NOT-SUPPORTED":
		err = errors.New("This instance of upsd does not support the requested feature. This is only used for TLS/SSL mode (STARTTLS) at the moment")
	case "FEATURE-NOT-CONFIGURED":
		err = errors.New("This instance of upsd hasn’t been configured properly to allow the requested feature to operate. This is also limited to STARTTLS for now")
	case "ALREADY-SSL-MODE":
		err = errors.New("TLS/SSL mode is already enabled on this connection, so upsd can’t start it again")
	case "DRIVER-NOT-CONNECTED":
		err = errors.New("upsd can’t perform the requested command, since the driver for that UPS is not connected. This usually means that the driver is not running, or if it is, the ups.conf is misconfigured")
	case "DATA-STALE":
		err = errors.New("upsd is connected to the driver for the UPS, but that driver isn’t providing regular updates or has specifically marked the data as stale. upsd refuses to provide variables on stale units to avoid false readings. This generally means that the driver is running, but it has lost communications with the hardware. Check the physical connection to the equipment")
	case "ALREADY-LOGGED-IN":
		err = errors.New("The client already sent LOGIN for a UPS and can’t do it again. There is presently a limit of one LOGIN record per connection")
	case "INVALID-PASSWORD":
		err = errors.New("The client sent an invalid PASSWORD - perhaps an empty one")
	case "ALREADY-SET-PASSWORD":
		err = errors.New("The client already set a PASSWORD and can’t set another. This also should never happen with normal NUT clients")
	case "INVALID-USERNAME":
		err = errors.New("The client sent an invalid USERNAME")
	case "ALREADY-SET-USERNAME":
		err = errors.New("The client has already set a USERNAME, and can’t set another. This should never happen with normal NUT clients")
	case "USERNAME-REQUIRED":
		err = errors.New("The requested command requires a username for authentication, but the client hasn’t set one")
	case "PASSWORD-REQUIRED":
		err = errors.New("The requested command requires a passname for authentication, but the client hasn’t set one")
	case "UNKNOWN-COMMAND":
		err = errors.New("upsd doesn’t recognize the requested command")
	case "INVALID-VALUE":
		err = errors.New("The value specified in the request is not valid. This usually applies to a SET of an ENUM type which is using a value which is not in the list of allowed values")
	default:
		err = errors.New("Unknown error code")
	}

	return err
}
