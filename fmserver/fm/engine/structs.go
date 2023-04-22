package engine

import "footmanager/fmcommon"

type SimulationPipeline interface {
	Simulate(dbCtx *fmcommon.DatabaseContext, match *fmcommon.MatchSimulation)
}

type Simulation interface {
	Apply(dbCtx *fmcommon.DatabaseContext, match *fmcommon.MatchSimulation)
}
