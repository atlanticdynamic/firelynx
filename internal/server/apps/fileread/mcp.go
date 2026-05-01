package fileread

import (
	"context"
	"errors"

	mcpio "github.com/robbyt/mcp-io"
)

// MCPToolName returns the default tool name used when no user override is set.
func (a *App) MCPToolName() string {
	return "fileread"
}

// MCPToolDescription returns the description shown to MCP clients.
func (a *App) MCPToolDescription() string {
	return "Read a file from the configured base directory"
}

// MCPToolOption returns the mcp-io option that registers this app as an MCP
// tool with input/output schemas auto-generated from Request / Response.
func (a *App) MCPToolOption(name string) mcpio.Option {
	return mcpio.WithTool(
		name,
		a.MCPToolDescription(),
		a.filereadToolFunc,
	)
}

// inputErrors are the readFile sentinel errors caused by bad client input;
// everything else (missing/unusable base directory, generic I/O failures)
// is a server-side problem that the LLM cannot fix by adjusting its call.
var inputErrors = []error{
	errMissingPath,
	errAbsolutePath,
	errDirectoryTraversal,
	errSymlinkEscape,
	errFileNotFound,
}

func (a *App) filereadToolFunc(
	_ context.Context,
	_ mcpio.RequestContext,
	input Request,
) (Response, error) {
	content, err := a.readFile(input.Path)
	if err != nil {
		for _, sentinel := range inputErrors {
			if errors.Is(err, sentinel) {
				return Response{}, mcpio.ValidationError(err.Error())
			}
		}
		return Response{}, mcpio.ProcessingError(err.Error())
	}

	return Response{Content: content}, nil
}
