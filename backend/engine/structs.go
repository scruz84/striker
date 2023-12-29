package engine

import "github.com/scruz84/striker/common"

type SimulationPipeline interface {
	Simulate(dbCtx *common.DatabaseContext, match *common.MatchSimulation)
}

type Simulation interface {
	Apply(dbCtx *common.DatabaseContext, match *common.MatchSimulation)
}
