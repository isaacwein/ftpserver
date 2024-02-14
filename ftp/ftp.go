// Description: FTP package
// This package contains the FTP server  implementations
// It also contains the FTP status codes and commands
// It is used to create a simple FTP server
// it also contains a file system interface and a local file system implementation for the FTP server

package ftp

// StatusCode is a type for FTP status codes
type StatusCode = int

const (
	// Informational codes (1xx)
	StatusRestartMarkerReply        StatusCode = 110 // Restart marker reply
	StatusServiceReadyInMinutes     StatusCode = 120 // Service ready in nnn minutes
	StatusDataConnectionAlreadyOpen StatusCode = 125 // Data connection already open; transfer starting
	StatusFileStatusOK              StatusCode = 150 // File status okay; about to open data connection

	// Success codes (2xx)
	StatusCommandOK                       StatusCode = 200 // Command okay
	StatusCommandNotImplemented           StatusCode = 202 // Command not implemented, superfluous at this site
	StatusSystemStatus                    StatusCode = 211 // System status, or system help reply
	StatusDirectoryStatus                 StatusCode = 212 // Directory status
	StatusFileStatus                      StatusCode = 213 // File status
	StatusHelpMessage                     StatusCode = 214 // Help message
	StatusNameSystemType                  StatusCode = 215 // NAME system type, where NAME is an official system name from the list in the Assigned Numbers document
	StatusServiceReadyForNewUser          StatusCode = 220 // Service ready for new user
	StatusServiceClosingControlConnection StatusCode = 221 // Service closing control connection
	StatusDataConnectionOpen              StatusCode = 225 // Data connection open; no transfer in progress
	StatusClosingDataConnection           StatusCode = 226 // Closing data connection; requested file action successful
	StatusEnteringPassiveMode             StatusCode = 227 // Entering Passive Mode (h1,h2,h3,h4,p1,p2)
	StatusEnteringLongPassiveMode         StatusCode = 228 // Entering Long Passive Mode (long address, port)
	StatusEnteringExtendedPassiveMode     StatusCode = 229 // Entering Extended Passive Mode (|||port|)
	StatusUserLoggedIn                    StatusCode = 230 // User logged in, proceed
	StatusUserAuthorized                  StatusCode = 232 // User logged in, authorized by security data exchange
	StatusSecurityExchangeOK              StatusCode = 234 // Server accepts authentication method/security mechanism
	StatusFileActionOK                    StatusCode = 250 // Requested file action okay, completed
	StatusPathnameCreated                 StatusCode = 257 // "PATHNAME" created

	// Transient Negative Completion codes (3xx)
	StatusCommandNotImplementedSuperfluous StatusCode = 331 // User name okay, need password
	StatusNeedAccountForLogin              StatusCode = 332 // Need account for login
	StatusFileActionPending                StatusCode = 350 // Requested file action pending further information
	// Transient Negative Completion codes (4xx)
	StatusServiceNotAvailable             StatusCode = 421 // Service not available, closing control connection
	StatusCantOpenDataConnection          StatusCode = 425 // Can't open data connection
	StatusConnectionClosedTransferAborted StatusCode = 426 // Connection closed; transfer aborted
	StatusInvalidUsernameOrPassword       StatusCode = 430 // Invalid username or password
	StatusNeedResourceToProcessSecurity   StatusCode = 431 // Need account for storing files
	StatusRequestedHostUnavailable        StatusCode = 434 // Requested host unavailable
	StatusRequestedFileActionNotTaken     StatusCode = 450 // Requested file action not taken
	StatusLocalProcessingError            StatusCode = 451 // Requested action aborted: local error in processing
	StatusInsufficientStorage             StatusCode = 452 // Requested action not taken; insufficient storage space
	// Permanent Negative Completion codes (5xx)
	StatusSyntaxError                   StatusCode = 500 // Syntax error, command unrecognized
	StatusSyntaxErrorInParameters       StatusCode = 501 // Syntax error in parameters or arguments
	StatusSyntaxErrorNotImplemented     StatusCode = 502 // Command not implemented
	StatusBadSequenceOfCommands         StatusCode = 503 // Bad sequence of commands
	StatusCommandNotImplementedForParam StatusCode = 504 // Command not implemented for that parameter
	StatusNotLoggedIn                   StatusCode = 530 // Not logged in
	StatusNeedAccountForStoringFiles    StatusCode = 532 // Need account for storing files
	StatusFileUnavailable               StatusCode = 550 // Requested action not taken; File unavailable
	StatusPageTypeUnknown               StatusCode = 551 // Requested action aborted: page type unknown
	StatusExceededStorageAllocation     StatusCode = 552 // Requested file action aborted; exceeded storage allocation
	StatusFileNameNotAllowed            StatusCode = 553 // Requested action not taken; file name not allowed
	// Protected reply codes (6xx)
	StatusIntegrityProtected                StatusCode = 631 // Integrity protected reply
	StatusConfidentialityIntegrityProtected StatusCode = 632 // Confidentiality and integrity protected reply
	StatusConfidentialityProtected          StatusCode = 633 // Confidentiality protected reply

	WinsockConnectionResetByPeer StatusCode = 10054 // Connection reset by peer
	WinsockCannotConnect         StatusCode = 10060 // Cannot connect to remote server
	WinsockConnectionRefused     StatusCode = 10061 // Connection refused
	WinsockNoRouteToHost         StatusCode = 10065 // No route to host
	WinsockDirectoryNotEmpty     StatusCode = 10066 // Directory not empty
	WinsockTooManyUsers          StatusCode = 10068 // Too many users
)

var statusText = map[StatusCode]string{
	110:   "StatusRestartMarkerReply",
	120:   "StatusServiceReadyInMinutes",
	125:   "StatusDataConnectionAlreadyOpen",
	150:   "StatusFileStatusOK",
	200:   "StatusCommandOK",
	202:   "StatusCommandNotImplemented",
	211:   "StatusSystemStatus",
	212:   "StatusDirectoryStatus",
	213:   "StatusFileStatus",
	214:   "StatusHelpMessage",
	215:   "StatusNameSystemType",
	220:   "StatusServiceReadyForNewUser",
	221:   "StatusServiceClosingControlConnection",
	225:   "StatusDataConnectionOpen",
	226:   "StatusClosingDataConnection",
	227:   "StatusEnteringPassiveMode",
	228:   "StatusEnteringLongPassiveMode",
	229:   "StatusEnteringExtendedPassiveMode",
	230:   "StatusUserLoggedIn",
	232:   "StatusUserAuthorized",
	234:   "StatusSecurityExchangeOK",
	250:   "StatusFileActionOK",
	257:   "StatusPathnameCreated",
	331:   "StatusCommandNotImplementedSuperfluous",
	332:   "StatusNeedAccountForLogin",
	350:   "StatusFileActionPending",
	421:   "StatusServiceNotAvailable",
	425:   "StatusCantOpenDataConnection",
	426:   "StatusConnectionClosedTransferAborted",
	430:   "StatusInvalidUsernameOrPassword",
	431:   "StatusNeedResourceToProcessSecurity",
	434:   "StatusRequestedHostUnavailable",
	450:   "StatusRequestedFileActionNotTaken",
	451:   "StatusLocalProcessingError",
	452:   "StatusInsufficientStorage",
	500:   "StatusSyntaxError",
	501:   "StatusSyntaxErrorInParameters",
	502:   "StatusSyntaxErrorNotImplemented",
	503:   "StatusBadSequenceOfCommands",
	504:   "StatusCommandNotImplementedForParam",
	530:   "StatusNotLoggedIn",
	532:   "StatusNeedAccountForStoringFiles",
	550:   "StatusFileUnavailable",
	551:   "StatusPageTypeUnknown",
	552:   "StatusExceededStorageAllocation",
	553:   "StatusFileNameNotAllowed",
	631:   "StatusIntegrityProtected",
	632:   "StatusConfidentialityIntegrityProtected",
	633:   "StatusConfidentialityProtected",
	10054: "WinsockConnectionResetByPeer",
	10060: "WinsockCannotConnect",
	10061: "WinsockConnectionRefused",
	10065: "WinsockNoRouteToHost",
	10066: "WinsockDirectoryNotEmpty",
	10068: "WinsockTooManyUsers",
}

func StatusText(code int) string {
	return statusText[code]
}

type Command = string

const (
	// Authentication and User Commands
	USER Command = "USER" // Send username
	PASS Command = "PASS" // Send password
	ACCT Command = "ACCT" // Send account information (rarely used)

	// Transfer Parameter Commands
	TYPE Command = "TYPE" // Set data transfer type (ASCII/Binary)
	MODE Command = "MODE" // Set data transfer mode (Stream/Block/Compressed)
	STRU Command = "STRU" // Set file structure  (File/Record/Page)

	// FTP Service Commands
	RETR Command = "RETR" // Retrieve a file
	STOR Command = "STOR" // Store a file
	STOU Command = "STOU" // Store a file with a unique name
	APPE Command = "APPE" // Append to a file
	ALLO Command = "ALLO" // Allocate storage (often unused)
	REST Command = "REST" // Restart an interrupted transfer
	RNFR Command = "RNFR" // Rename from (start the rename process)
	RNTO Command = "RNTO" // Rename to   (finish the rename process)
	ABOR Command = "ABOR" // Abort an active transfer
	DELE Command = "DELE" // Delete a file
	CWD  Command = "CWD"  // Change working directory
	CDUP Command = "CDUP" // Change to parent directory
	MKD  Command = "MKD"  // Make directory
	XMKD Command = "XMKD" // Make directory (extended version)
	RMD  Command = "RMD"  // Remove directory
	XRMD Command = "XRMD" // Remove directory (extended version)

	// Informational Commands
	PWD  Command = "PWD"  // Print working directory
	LIST Command = "LIST" // List directory contents
	NLST Command = "NLST" // Get concise list of filenames
	SITE Command = "SITE" // Send site-specific commands (varies between servers)
	SYST Command = "SYST" // Get operating system type
	STAT Command = "STAT" // Get server status
	HELP Command = "HELP" // Get help

	// Miscellaneous
	NOOP Command = "NOOP" // No operation (often used to keep connections alive)
	QUIT Command = "QUIT" // Disconnect from the server
)
