package sftp

// Initialize (INIT): This is the first request sent by the client to start an SFTP session.
//	The server responds with its supported version.
// Open (OPEN): Opens a file for reading, writing, or both. The server returns a handle if successful,
//	which is used in subsequent operations on the file.
// Close (CLOSE): Closes a file or directory identified by a handle previously returned by the server.
// Read (READ): Reads data from a file identified by a handle. The request includes the handle, offset, and the number of bytes to read.
// Write (WRITE): Writes data to a file identified by a handle. The request includes the handle, offset, and the data to write.
// List (READDIR): Reads a directory's contents, returning a list of files and their attributes.
// Stat (STAT): Retrieves attributes for a file or directory.
// FStat (FSTAT): Retrieves attributes for an open file or directory identified by a handle.
// SetStat (SETSTAT): Sets attributes for a file or directory.
// FSetStat (FSETSTAT): Sets attributes for an open file or directory identified by a handle.
// Remove (REMOVE): Deletes a file.
// Rename (RENAME): Renames a file or directory.
// Make Directory (MKDIR): Creates a new directory.
// Remove Directory (RMDIR): Removes a directory.
// RealPath (REALPATH): Canonicalizes the server-side path.
// SymLink (SYMLINK): Creates a symbolic link.
// ReadLink (READLINK): Reads the target of a symbolic link.
// Version (VERSION): Used to negotiate the protocol version. The client sends its highest supported version,
//	and the server responds with the version they will use.

const (
	MethodInit     = "INIT"
	MethodOpen     = "OPEN"
	MethodClose    = "CLOSE"
	MethodRead     = "READ"
	MethodWrite    = "WRITE"
	MethodList     = "READDIR"
	MethodStat     = "STAT"
	MethodFStat    = "FSTAT"
	MethodSetStat  = "SETSTAT"
	MethodFSetStat = "FSETSTAT"
	MethodRemove   = "REMOVE"
	MethodRename   = "RENAME"
	MethodMkdir    = "MKDIR"
	MethodRmdir    = "RMDIR"
	MethodRealPath = "REALPATH"
	MethodSymLink  = "SYMLINK"
	MethodReadLink = "READLINK"
	MethodVersion  = "VERSION"
)
