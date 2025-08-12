package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func CheckStartOrEnd(line string, pendingStart bool, pendingEnd bool, g *Graph) (isStart, isEnd bool) {

	if line == "##start" {
		if pendingStart || g.Start != nil {
			fmt.Println("ERROR: invalid data format")
			fmt.Println("Reason: duplicate start")
			os.Exit(1)
		}
		return true, false
	} else if line == "##end" {
		if pendingEnd || g.End != nil {
			fmt.Println("ERROR: invalid data format")
			fmt.Println("Reason: duplicate end")
			os.Exit(1)
		}
		return false, true
	}

	return false, false
}

func CheckAnts(line string) (int, bool) {
	ants, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || ants <= 0 {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: invalid ants count")
		os.Exit(1)
	}
	if ants > MaxAnts {
		fmt.Println("ERROR: ant limit exceeded")
		fmt.Println("Reason: ant count is more than " + strconv.Itoa(MaxAnts))
		os.Exit(1)
	}
	return ants, true
}

func CheckRoom(pendingStart, pendingEnd bool, g *Graph, fields []string, coords map[[2]int]bool) (bool, bool, bool, *Graph) {
	name := fields[0]
	if strings.HasPrefix(name, "L") || strings.HasPrefix(name, "#") {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: invalid room name '" + name + "'")
		os.Exit(1)
	}
	if _, ok := g.Rooms[name]; ok {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: duplicate room name '" + name + "'")
		os.Exit(1)
	}
	x, err1 := strconv.Atoi(fields[1])
	y, err2 := strconv.Atoi(fields[2])
	if err1 != nil || err2 != nil {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: invalid room line")
		os.Exit(1)
	}
	if coords[[2]int{x, y}] {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: duplicate coordinates " + fields[1] + " " + fields[2])
		os.Exit(1)
	}
	coords[[2]int{x, y}] = true
	r := &Room{Name: name, X: x, Y: y}
	g.Rooms[name] = r
	if pendingStart {
		g.Start = r
		pendingStart = false
	}
	if pendingEnd {
		g.End = r
		pendingEnd = false
	}
	return pendingStart, pendingEnd, coords[[2]int{x, y}], g

}

func CheckLink(linkSeen map[string]struct{}, links [][2]string, a, b string) ([][2]string, map[string]struct{}) {
	if a == b {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: self-loop link " + a + "-" + b)
		os.Exit(1)
	}
	if b < a {
		a, b = b, a
	}
	key := a + "-" + b
	if _, ok := linkSeen[key]; ok {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: duplicate link " + key)
		os.Exit(1)
	}
	linkSeen[key] = struct{}{}
	links = append(links, [2]string{a, b})
	return links, linkSeen
}
