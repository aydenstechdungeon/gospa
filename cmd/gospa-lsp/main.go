// Package main implements the GoSPA Language Server Protocol.
package main

import (
	"log"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

var (
	handler protocol.Handler
	version = "0.0.1"
)

func main() {
	handler = protocol.Handler{
		Initialize:             initialize,
		Initialized:            initialized,
		Shutdown:               shutdown,
		SetTrace:               setTrace,
		TextDocumentDidOpen:    didOpen,
		TextDocumentDidChange:  didChange,
		TextDocumentDidSave:    didSave,
		TextDocumentHover:      hover,
		TextDocumentCompletion: completion,
	}

	s := server.NewServer(&handler, "gospa-lsp", false)
	err := s.RunStdio()
	if err != nil {
		log.Fatalf("error: %s", err)
	}
}

func initialize(_ *glsp.Context, _ *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()
	capabilities.TextDocumentSync = protocol.TextDocumentSyncKindFull
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"$"},
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "gospa-lsp",
			Version: &version,
		},
	}, nil
}

func initialized(_ *glsp.Context, _ *protocol.InitializedParams) error {
	return nil
}

func shutdown(_ *glsp.Context) error {
	protocol.SetTraceValue(protocol.TraceValueOff)
	return nil
}

func setTrace(_ *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func didOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	validate(context, params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

func didChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			validate(context, params.TextDocument.URI, c.Text)
		}
	}
	return nil
}

func didSave(_ *glsp.Context, _ *protocol.DidSaveTextDocumentParams) error {
	return nil
}

func validate(context *glsp.Context, uri string, text string) {
	diagnostics := []protocol.Diagnostic{}

	_, err := sfc.Parse(text)
	if err != nil {
		// Try to find if the error message contains info about where it happened.
		// For now, we'll just put it at 0:0 since sfc.Parse doesn't return offset yet for errors.
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 1},
			},
			Message:  err.Error(),
			Severity: &diagnosticsSeverityError,
		})
	}

	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})
}

var diagnosticsSeverityError = protocol.DiagnosticSeverityError

func hover(_ *glsp.Context, _ *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, nil
}

func completion(_ *glsp.Context, _ *protocol.CompletionParams) (any, error) {
	items := []protocol.CompletionItem{
		{
			Label:            "$state",
			Kind:             &completionItemKindFunction,
			Detail:           ptr("Declare reactive state"),
			InsertText:       ptr("$state($1)"),
			InsertTextFormat: &insertTextFormatSnippet,
		},
		{
			Label:            "$derived",
			Kind:             &completionItemKindFunction,
			Detail:           ptr("Declare derived reactive state"),
			InsertText:       ptr("$derived($1)"),
			InsertTextFormat: &insertTextFormatSnippet,
		},
		{
			Label:            "$effect",
			Kind:             &completionItemKindFunction,
			Detail:           ptr("Declare side effect"),
			InsertText:       ptr("$effect(func() {\n\t$1\n})"),
			InsertTextFormat: &insertTextFormatSnippet,
		},
		{
			Label:            "$props",
			Kind:             &completionItemKindFunction,
			Detail:           ptr("Access component props"),
			InsertText:       ptr("$props()"),
			InsertTextFormat: &insertTextFormatSnippet,
		},
	}

	return items, nil
}

var (
	completionItemKindFunction = protocol.CompletionItemKindFunction
	insertTextFormatSnippet    = protocol.InsertTextFormatSnippet
)

func ptr[T any](v T) *T {
	return &v
}
