package utils

// SimulateMulti runs the ant simulation for multiple paths.
func SimulateMulti(g *Graph, paths [][]*Room) []string {
	if len(paths) == 0 {
		return nil
	}
	// plan tells which path each ant will take
	plan := assignPaths(paths, g.Ants)
	// wait holds ants waiting to start on each path
	wait := make([][]int, len(paths))
	for ant, p := range plan {
		wait[p] = append(wait[p], ant)
	}
	loc := make([]int, len(plan))    // current step in path for each ant
	going := make([]bool, len(plan)) // has the ant started
	busy := map[*Room]int{}          // rooms currently occupied
	done := 0                        // number of ants finished
	var out []string                 // output lines
	for done < len(plan) {
		moves := moveAnts(g, paths, loc, going, busy, &done, plan)
		moves = append(moves, startAnts(g, paths, wait, going, loc, busy, &done)...)
		if len(moves) > 0 {
			out = append(out, formatMoves(moves))
		}
	}
	return out
}
