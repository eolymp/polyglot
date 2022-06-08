package types

import (
	"context"
	"github.com/eolymp/contracts/go/eolymp/atlas"
	"github.com/eolymp/contracts/go/eolymp/executor"
	"github.com/eolymp/contracts/go/eolymp/keeper"
	"github.com/eolymp/contracts/go/eolymp/typewriter"
)

type Importer interface {
	GetVerifier() (*executor.Verifier, error)

	HasInteractor() bool
	GetInteractor() (*executor.Interactor, error)

	GetStatements(context.Context, *typewriter.TypewriterService, string) ([]*atlas.Statement, error)

	GetSolutions() ([]*atlas.Solution, error)

	GetTestsets(*keeper.KeeperService) ([]*Group, error)

	GetTemplates(pid *string) ([]*atlas.Template, error)
}

type Group struct {
	Testset *atlas.Testset
	Tests   []*atlas.Test
	Name    uint32
}
