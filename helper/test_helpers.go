package helper

import (
	abci "github.com/cometbft/cometbft/abci/types"
)

type TestOpts struct {
	app     abci.Application
	chainID string
}

func NewTestOpts(app abci.Application, chainID string) *TestOpts {
	return &TestOpts{
		app:     app,
		chainID: chainID,
	}
}

func (t *TestOpts) SetApplication(app abci.Application) {
	t.app = app
}

func (t *TestOpts) GetApplication() abci.Application {
	return t.app
}

func (t *TestOpts) SetChainID(chainID string) {
	t.chainID = chainID
}

func (t *TestOpts) GetChainID() string {
	return t.chainID
}
