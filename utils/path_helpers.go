package utils

// These helpers help assign ants to paths in a simple way.

// getLens returns how many steps are in each path (rooms minus one).
func getLens(paths [][]*Room) []int {
	lens := make([]int, len(paths))
	for i, path := range paths {
		// each path length is number of rooms minus the start room
		lens[i] = len(path) - 1
	}
	return lens
}

// countStarts figures out how many ants can start on each path.
func countStarts(lens []int, turns int) []int {
	starts := make([]int, len(lens))
	for i, l := range lens {
		// the later the path, the fewer ants can use it
		c := turns - l + 1
		if c < 0 {
			c = 0
		}
		starts[i] = c
	}
	return starts
}

// trimStarts makes sure we do not start more ants than we have.
func trimStarts(starts []int, ants int) []int {
	total := 0
	for _, s := range starts {
		total += s
	}
	for i := len(starts) - 1; total > ants && i >= 0; i-- {
		if starts[i] > 0 {
			take := starts[i]
			if take > total-ants {
				take = total - ants
			}
			starts[i] -= take
			total -= take
		}
	}
	return starts
}

// planOrder builds the order of path indices for ant departure.
func planOrder(starts []int) []int {
	var order []int
	active := true
	n := len(starts)
	for active {
		active = false
		for i := 0; i < n; i++ {
			if starts[i] > 0 {
				order = append(order, i)
				starts[i]--
				active = true
			}
		}
	}
	return order
}
