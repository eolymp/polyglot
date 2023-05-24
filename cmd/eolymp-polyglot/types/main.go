package types

import (
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/executor"
)

type Importer interface {
	GetVerifier() (*executor.Verifier, error)

	HasInteractor() bool
	GetInteractor() (*executor.Interactor, error)

	GetStatements(string) ([]*atlas.Statement, error)

	GetSolutions() ([]*atlas.Editorial, error)

	GetTestsets() ([]*Group, error)

	GetTemplates(*string) ([]*atlas.Template, error)

	GetAttachments(*string) ([]*atlas.Attachment, error)
}

type Group struct {
	Testset *atlas.Testset
	Tests   []*atlas.Test
	Name    uint32
}
