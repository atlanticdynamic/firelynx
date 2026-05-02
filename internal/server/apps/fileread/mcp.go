package fileread

import (
	"context"
	"errors"
	"io"

	mcpio "github.com/robbyt/mcp-io"
)

// stringFileReader is what filereadToolFunc needs from a resolved file.
// *ResolvedFile satisfies it.
type stringFileReader interface {
	io.Closer
	ReadAllString() (string, error)
}

// Request defines the typed input parameters for file read requests.
type Request struct {
	Path string `json:"path" jsonschema:"Path to read, relative to base directory"`
}

// Response defines the typed output structure for file read responses.
type Response struct {
	Content string `json:"content"         jsonschema:"Contents of the requested file"`
	Error   string `json:"error,omitempty"`
}

// MCPToolName returns the default tool name used when no user override is set.
func (a *App) MCPToolName() string { return "fileread" }

// MCPToolDescription returns the description shown to MCP clients.
func (a *App) MCPToolDescription() string {
	return "Read a file from the configured base directory"
}

// MCPToolOption returns the mcp-io option that registers this app as an MCP
// tool with input/output schemas auto-generated from Request / Response.
func (a *App) MCPToolOption(name string) mcpio.Option {
	return mcpio.WithTool(name, a.MCPToolDescription(), a.filereadToolFunc)
}

// inputErrors are the ResolveFile sentinel errors caused by bad client
// input; everything else (missing/unusable base directory, generic I/O
// failures) is a server-side problem the LLM cannot fix by adjusting
// its call.
var inputErrors = []error{
	errMissingPath,
	errAbsolutePath,
	errDirectoryTraversal,
	errSymlinkEscape,
	errFileNotFound,
	errTargetIsDirectory,
}

// resolveForMCP is the App's hook to ResolveFile for the MCP path.
// Returning the interface keeps filereadToolFunc coupled only to the
// surface it needs.
func (a *App) resolveForMCP(requestedPath string) (stringFileReader, error) {
	return ResolveFile(a.baseDirectory, requestedPath, a.allowExternalSymlinks)
}

func (a *App) filereadToolFunc(
	_ context.Context,
	_ mcpio.RequestContext,
	input Request,
) (Response, error) {
	f, err := a.resolveForMCP(input.Path)
	if err != nil {
		for _, sentinel := range inputErrors {
			if errors.Is(err, sentinel) {
				return Response{}, mcpio.ValidationError(err.Error())
			}
		}
		return Response{}, mcpio.ProcessingError(err.Error())
	}
	defer func() {
		// Close error not actionable: the underlying *os.File is read-only
		// (no flush to fail) and the tool response is already shaped from
		// ReadAllString's outcome.
		_ = f.Close() //nolint:errcheck
	}()

	content, err := f.ReadAllString()
	if err != nil {
		return Response{}, mcpio.ProcessingError(err.Error())
	}
	return Response{Content: content}, nil
}
